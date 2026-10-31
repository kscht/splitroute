package main

import (
	"fmt"
	"log"
	"net/netip"
	"sort"
)

func cmdValidate(cfg *Config) error {
	log.Println("loading networks...")
	orgNets, err := readNets(cfg.Files.Networks)
	if err != nil {
		return fmt.Errorf("load org: %w", err)
	}
	optNets, err := readNets(cfg.Files.Optimized)
	if err != nil {
		return fmt.Errorf("load optimized: %w", err)
	}
	ruNets, err := readNets(ruCacheFile)
	if err != nil {
		return fmt.Errorf("load ru: %w", err)
	}
	exNets, err := loadExcluded(cfg)
	if err != nil {
		return err
	}
	log.Printf("org: %d, optimized: %d, ru: %d, excluded: %d", len(orgNets), len(optNets), len(ruNets), len(exNets))

	sortNets(optNets)
	sortNets(orgNets)

	optAddrs := addrs(optNets)
	orgAddrs := addrs(orgNets)

	ok := true

	// 1. Каждый org-CIDR должен быть покрыт каким-то optimized-CIDR.
	var uncovered []netip.Prefix
	for _, net := range orgNets {
		idx := sort.Search(len(optNets), func(i int) bool {
			return optAddrs[i].Compare(net.Addr()) > 0
		}) - 1
		if idx < 0 || !subnetOf(net, optNets[idx]) {
			uncovered = append(uncovered, net)
		}
	}
	if len(uncovered) > 0 {
		ok = false
		log.Printf("FAIL: %d org networks not covered by optimized", len(uncovered))
		for _, n := range uncovered[:min(5, len(uncovered))] {
			log.Printf("  uncovered: %s", n)
		}
	} else {
		log.Printf("OK: all %d org networks covered", len(orgNets))
	}

	// 2–3. Ни один optimized-CIDR не должен захватывать новые RU-сети и сети исключённых организаций.
	for _, chk := range []struct {
		name string
		nets []netip.Prefix
	}{{"RU", ruNets}, {"excluded", exNets}} {
		captured := newlyCaptured(chk.nets, optNets, optAddrs, orgNets, orgAddrs)
		if len(captured) > 0 {
			ok = false
			log.Printf("FAIL: %d %s networks newly captured", len(captured), chk.name)
			for _, n := range captured[:min(5, len(captured))] {
				log.Printf("  new %s: %s", chk.name, n)
			}
		} else {
			log.Printf("OK: no new %s networks captured", chk.name)
		}
	}

	if !ok {
		return fmt.Errorf("validation failed")
	}
	log.Println("validation passed")
	return nil
}

// newlyCaptured — сети из nets, покрытые оптимизированным набором, но не покрытые исходными org-сетями.
func newlyCaptured(nets, optNets []netip.Prefix, optAddrs []netip.Addr, orgNets []netip.Prefix, orgAddrs []netip.Addr) []netip.Prefix {
	var out []netip.Prefix
	for _, n := range nets {
		idx := sort.Search(len(optNets), func(i int) bool {
			return optAddrs[i].Compare(n.Addr()) > 0
		}) - 1
		if idx < 0 || !subnetOf(n, optNets[idx]) {
			continue
		}
		idx2 := sort.Search(len(orgNets), func(i int) bool {
			return orgAddrs[i].Compare(n.Addr()) > 0
		}) - 1
		if idx2 < 0 || !subnetOf(n, orgNets[idx2]) {
			out = append(out, n)
		}
	}
	return out
}

func addrs(nets []netip.Prefix) []netip.Addr {
	a := make([]netip.Addr, len(nets))
	for i, n := range nets {
		a[i] = n.Addr()
	}
	return a
}
