package main

import (
	"bufio"
	"fmt"
	"log"
	"net/netip"
	"os"
	"sort"
	"strings"
)

var privateNets []netip.Prefix

func init() {
	for _, s := range []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"100.64.0.0/10",
		"169.254.0.0/16",
		"127.0.0.0/8",
		"192.0.2.0/24",
		"198.51.100.0/24",
		"203.0.113.0/24",
		"224.0.0.0/4",
		"240.0.0.0/4",
	} {
		p, _ := netip.ParsePrefix(s)
		privateNets = append(privateNets, p.Masked())
	}
}

func cmdOptimize(cfg *Config) error {
	log.Println("loading networks...")
	orgNets, err := readNets(cfg.Files.Networks)
	if err != nil {
		return fmt.Errorf("load org: %w", err)
	}
	ruNets, err := readNets(ruCacheFile)
	if err != nil {
		return fmt.Errorf("load ru: %w", err)
	}
	log.Printf("org: %d, ru: %d", len(orgNets), len(ruNets))

	sortNets(ruNets)

	log.Println("optimizing...")
	result := optimizeNets(orgNets, ruNets)

	ratio := float64(len(orgNets)) / float64(max(1, len(result)))
	log.Printf("result: %d → %d (%.1fx)", len(orgNets), len(result), ratio)

	if err := writeNets(cfg.Files.Optimized, result); err != nil {
		return err
	}
	log.Printf("saved → %s", cfg.Files.Optimized)
	return nil
}

func readNets(path string) ([]netip.Prefix, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var nets []netip.Prefix
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		p, err := netip.ParsePrefix(line)
		if err != nil {
			continue
		}
		nets = append(nets, p.Masked())
	}
	return nets, sc.Err()
}

func optimizeNets(orgNets, ruNets []netip.Prefix) []netip.Prefix {
	nets := dedupe(orgNets)
	// Progressively expand to larger prefixes, then collapse at each step.
	for targetBits := 11; targetBits <= 22; targetBits++ {
		nets = expandAll(nets, targetBits, ruNets)
		nets = collapse(nets, ruNets)
	}
	return nets
}

// dedupe removes prefixes that are subnets of an earlier prefix in sorted order.
func dedupe(nets []netip.Prefix) []netip.Prefix {
	sortNets(nets)
	var out []netip.Prefix
	for _, p := range nets {
		if len(out) > 0 {
			last := out[len(out)-1]
			if last.Bits() <= p.Bits() && last.Contains(p.Addr()) {
				continue
			}
		}
		out = append(out, p)
	}
	return out
}

// expandAll tries to expand each prefix to targetBits if no RU or private IPs
// would be newly captured.
func expandAll(nets []netip.Prefix, targetBits int, ruNets []netip.Prefix) []netip.Prefix {
	out := make([]netip.Prefix, 0, len(nets))
	for _, p := range nets {
		if p.Bits() > targetBits {
			sup, _ := p.Addr().Prefix(targetBits)
			sup = sup.Masked()
			if !overlapsPrivate(sup) && !newRU(p, sup, ruNets) {
				out = append(out, sup)
				continue
			}
		}
		out = append(out, p)
	}
	return out
}

// collapse iteratively merges adjacent sibling prefixes until stable.
func collapse(nets []netip.Prefix, ruNets []netip.Prefix) []netip.Prefix {
	nets = dedupe(nets)
	for {
		next := mergePairs(nets, ruNets)
		next = dedupe(next)
		if len(next) == len(nets) {
			return next
		}
		nets = next
	}
}

// mergePairs does one pass of merging adjacent same-size sibling prefixes.
func mergePairs(nets []netip.Prefix, ruNets []netip.Prefix) []netip.Prefix {
	out := make([]netip.Prefix, 0, len(nets))
	i := 0
	for i < len(nets) {
		if i+1 < len(nets) {
			a, b := nets[i], nets[i+1]
			if a.Bits() == b.Bits() && a.Bits() > 0 {
				sup, _ := a.Addr().Prefix(a.Bits() - 1)
				sup = sup.Masked()
				if sup.Contains(a.Addr()) && sup.Contains(b.Addr()) &&
					!overlapsPrivate(sup) &&
					!newRUMerge(a, b, sup, ruNets) {
					out = append(out, sup)
					i += 2
					continue
				}
			}
		}
		out = append(out, nets[i])
		i++
	}
	return out
}

// newRU reports whether expanding from→to newly captures any RU network.
func newRU(from, to netip.Prefix, ruNets []netip.Prefix) bool {
	first, last := to.Addr(), lastAddr(to)
	idx := sort.Search(len(ruNets), func(i int) bool {
		return ruNets[i].Addr().Compare(first) >= 0
	})
	for _, ru := range ruNets[idx:] {
		if ru.Addr().Compare(last) > 0 {
			break
		}
		if subnetOf(ru, to) && !subnetOf(ru, from) {
			return true
		}
	}
	return false
}

// newRUMerge reports whether merging a+b→sup newly captures any RU network.
func newRUMerge(a, b, sup netip.Prefix, ruNets []netip.Prefix) bool {
	first, last := sup.Addr(), lastAddr(sup)
	idx := sort.Search(len(ruNets), func(i int) bool {
		return ruNets[i].Addr().Compare(first) >= 0
	})
	for _, ru := range ruNets[idx:] {
		if ru.Addr().Compare(last) > 0 {
			break
		}
		if subnetOf(ru, sup) && !subnetOf(ru, a) && !subnetOf(ru, b) {
			return true
		}
	}
	return false
}

func subnetOf(sub, sup netip.Prefix) bool {
	return sup.Contains(sub.Addr()) && sub.Bits() >= sup.Bits()
}

func overlapsPrivate(p netip.Prefix) bool {
	for _, priv := range privateNets {
		if p.Overlaps(priv) {
			return true
		}
	}
	return false
}

func sortNets(nets []netip.Prefix) {
	sort.Slice(nets, func(i, j int) bool {
		if c := nets[i].Addr().Compare(nets[j].Addr()); c != 0 {
			return c < 0
		}
		return nets[i].Bits() < nets[j].Bits()
	})
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
