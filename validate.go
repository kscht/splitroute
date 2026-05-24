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
	log.Printf("org: %d, optimized: %d, ru: %d", len(orgNets), len(optNets), len(ruNets))

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

	// 2. Ни один optimized-CIDR не должен захватывать новые RU-сети.
	var newRU []netip.Prefix
	for _, ru := range ruNets {
		// Есть ли optimized, содержащий эту RU-сеть?
		idx := sort.Search(len(optNets), func(i int) bool {
			return optAddrs[i].Compare(ru.Addr()) > 0
		}) - 1
		if idx < 0 || !subnetOf(ru, optNets[idx]) {
			continue
		}
		// Была ли она уже покрыта в оригинальных org-сетях?
		idx2 := sort.Search(len(orgNets), func(i int) bool {
			return orgAddrs[i].Compare(ru.Addr()) > 0
		}) - 1
		if idx2 < 0 || !subnetOf(ru, orgNets[idx2]) {
			newRU = append(newRU, ru)
		}
	}
	if len(newRU) > 0 {
		ok = false
		log.Printf("FAIL: %d RU networks newly captured", len(newRU))
		for _, n := range newRU[:min(5, len(newRU))] {
			log.Printf("  new RU: %s", n)
		}
	} else {
		log.Printf("OK: no new RU networks captured")
	}

	if !ok {
		return fmt.Errorf("validation failed")
	}
	log.Println("validation passed")
	return nil
}

func addrs(nets []netip.Prefix) []netip.Addr {
	a := make([]netip.Addr, len(nets))
	for i, n := range nets {
		a[i] = n.Addr()
	}
	return a
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
