package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
)

func cmdRoutes(cfg *Config) error {
	nets, err := readNets(cfg.Files.Optimized)
	if err != nil {
		return fmt.Errorf("load optimized: %w", err)
	}

	f, err := os.Create(cfg.Files.Routes)
	if err != nil {
		return err
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	for _, p := range nets {
		fmt.Fprintf(w, "ip route %s %s %s %s %s\n",
			p.Addr(), cidrMask(p.Bits()),
			cfg.Routing.GatewayIP, cfg.Routing.GatewayName, cfg.Routing.RouteType)
	}
	if err := w.Flush(); err != nil {
		return err
	}

	log.Printf("generated %d routes → %s", len(nets), cfg.Files.Routes)

	if cfg.Files.CIDR != "" {
		if err := writeNets(cfg.Files.CIDR, nets); err != nil {
			return err
		}
		log.Printf("saved CIDRs → %s", cfg.Files.CIDR)
	}
	return nil
}

func cidrMask(bits int) string {
	m := ^uint32(0) << (32 - bits)
	return fmt.Sprintf("%d.%d.%d.%d", m>>24, (m>>16)&0xff, (m>>8)&0xff, m&0xff)
}
