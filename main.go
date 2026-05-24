package main

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

type Config struct {
	API     APIConfig     `toml:"api"`
	Files   FilesConfig   `toml:"files"`
	Routing RoutingConfig `toml:"routing"`
}

type APIConfig struct {
	Token string `toml:"token"`
}

type FilesConfig struct {
	OrgList   string `toml:"org_list"`
	Networks  string `toml:"networks_output"`
	Optimized string `toml:"optimized_output"`
	Routes    string `toml:"routes_output"`
	CIDR      string `toml:"cidr_output"`
}

type RoutingConfig struct {
	GatewayIP   string `toml:"gateway_ip"`
	GatewayName string `toml:"gateway_name"`
	RouteType   string `toml:"route_type"`
}

func loadConfig(path string) (*Config, error) {
	var cfg Config
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	return &cfg, nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: splitroute <fetch|optimize|routes|validate|all|lookup <ip>>")
		os.Exit(1)
	}

	cfg, err := loadConfig("config.toml")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	cmd := os.Args[1]
	switch cmd {
	case "fetch":
		err = cmdFetch(cfg)
	case "optimize":
		err = cmdOptimize(cfg)
	case "routes":
		err = cmdRoutes(cfg)
	case "validate":
		err = cmdValidate(cfg)
	case "lookup":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: splitroute lookup <ip>")
			os.Exit(1)
		}
		err = cmdLookup(cfg, os.Args[2])
	case "all":
		for _, fn := range []func(*Config) error{cmdFetch, cmdOptimize, cmdRoutes} {
			if err = fn(cfg); err != nil {
				break
			}
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
