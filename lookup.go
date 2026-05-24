package main

import (
	"fmt"
	"net/netip"
	"sort"
)

func cmdLookup(cfg *Config, ipStr string) error {
	addr, err := netip.ParseAddr(ipStr)
	if err != nil {
		return fmt.Errorf("invalid IP: %w", err)
	}

	nets, err := readNets(cfg.Files.Optimized)
	if err != nil {
		return fmt.Errorf("load optimized: %w", err)
	}
	sortNets(nets)

	addrs := addrs(nets)
	idx := sort.Search(len(nets), func(i int) bool {
		return addrs[i].Compare(addr) > 0
	}) - 1

	if idx >= 0 && nets[idx].Contains(addr) {
		fmt.Printf("HIT  %s  →  %s\n", addr, nets[idx])
		return nil
	}
	fmt.Printf("MISS %s\n", addr)
	return nil
}
