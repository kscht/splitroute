package main

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/netip"
	"os"
	"strings"
	"time"
)

const (
	cacheDir    = ".cache"
	dbCacheFile = ".cache/country_asn.json.gz"
	ruCacheFile = ".cache/ru_networks.txt"
	cacheTTL    = 24 * time.Hour
	ipinfoURL   = "https://ipinfo.io/data/free/country_asn.json.gz"
)

type ipRecord struct {
	StartIP  string `json:"start_ip"`
	EndIP    string `json:"end_ip"`
	Country  string `json:"country"`
	ASDomain string `json:"as_domain"`
}

func cmdFetch(cfg *Config) error {
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return err
	}

	if !cacheValid(dbCacheFile) {
		log.Println("downloading ipinfo database...")
		if err := downloadDB(cfg.API.Token); err != nil {
			return fmt.Errorf("download: %w", err)
		}
	} else {
		log.Println("using cached database")
	}

	orgs, err := loadOrgList(cfg.Files.OrgList)
	if err != nil {
		return err
	}
	log.Printf("organizations: %d", len(orgs))

	orgNets, ruNets, err := parseDB(orgs)
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}
	log.Printf("found %d org CIDRs, %d RU CIDRs", len(orgNets), len(ruNets))

	if err := writeNets(cfg.Files.Networks, orgNets); err != nil {
		return err
	}
	if err := writeNets(ruCacheFile, ruNets); err != nil {
		return err
	}
	log.Printf("saved → %s", cfg.Files.Networks)
	return nil
}

func cacheValid(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && time.Since(fi.ModTime()) < cacheTTL
}

func downloadDB(token string) error {
	req, err := http.NewRequest("GET", ipinfoURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	f, err := os.Create(dbCacheFile)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	return err
}

func loadOrgList(path string) (map[string]bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	orgs := make(map[string]bool)
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if line := strings.TrimSpace(strings.ToLower(sc.Text())); line != "" {
			orgs[line] = true
		}
	}
	return orgs, sc.Err()
}

func parseDB(orgs map[string]bool) (orgNets, ruNets []netip.Prefix, err error) {
	f, err := os.Open(dbCacheFile)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return nil, nil, err
	}
	defer gz.Close()

	sc := bufio.NewScanner(gz)
	sc.Buffer(make([]byte, 1<<20), 1<<20)

	var rec ipRecord
	for sc.Scan() {
		b := sc.Bytes()
		if len(b) == 0 {
			continue
		}
		if err := json.Unmarshal(b, &rec); err != nil {
			continue
		}
		if !strings.Contains(rec.StartIP, ".") {
			continue // IPv6 — skip
		}
		cidrs, err := rangeToCIDRs(rec.StartIP, rec.EndIP)
		if err != nil {
			continue
		}
		if orgs[strings.ToLower(rec.ASDomain)] {
			orgNets = append(orgNets, cidrs...)
		}
		if rec.Country == "RU" {
			ruNets = append(ruNets, cidrs...)
		}
	}
	return orgNets, ruNets, sc.Err()
}

func rangeToCIDRs(startStr, endStr string) ([]netip.Prefix, error) {
	start, err := netip.ParseAddr(startStr)
	if err != nil {
		return nil, err
	}
	end, err := netip.ParseAddr(endStr)
	if err != nil {
		return nil, err
	}
	return summarizeRange(start, end), nil
}

// summarizeRange converts an inclusive IP range to a minimal set of CIDR prefixes.
func summarizeRange(start, end netip.Addr) []netip.Prefix {
	var result []netip.Prefix
	for start.IsValid() && start.Compare(end) <= 0 {
		// Find the largest prefix rooted at start that fits within end.
		bits := 32
		for bits > 0 {
			p, _ := start.Prefix(bits - 1)
			p = p.Masked()
			if p.Addr() != start {
				break
			}
			if lastAddr(p).Compare(end) > 0 {
				break
			}
			bits--
		}
		p, _ := start.Prefix(bits)
		p = p.Masked()
		result = append(result, p)
		last := lastAddr(p)
		if last.Compare(end) >= 0 {
			break
		}
		start = last.Next()
	}
	return result
}

// lastAddr returns the broadcast (last) address of a prefix.
func lastAddr(p netip.Prefix) netip.Addr {
	a := p.Addr().As4()
	n := uint32(a[0])<<24 | uint32(a[1])<<16 | uint32(a[2])<<8 | uint32(a[3])
	n |= ^uint32(0) >> p.Bits()
	return netip.AddrFrom4([4]byte{byte(n >> 24), byte(n >> 16), byte(n >> 8), byte(n)})
}

func writeNets(path string, nets []netip.Prefix) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	for _, n := range nets {
		fmt.Fprintln(w, n)
	}
	return w.Flush()
}
