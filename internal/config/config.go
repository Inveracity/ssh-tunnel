package config

import (
	"errors"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/fatih/color"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclparse"
)

type Configuration struct {
	Tunnels []Tunnel `hcl:"tunnel,block"`
}

type Tunnel struct {
	Name   string `hcl:"name,optional"`
	User   string `hcl:"user"`
	Local  Local  `hcl:"local,block"`
	Remote Remote `hcl:"remote,block"`
}

type Local struct {
	Port string   `hcl:"port"`
	Cmd  []string `hcl:"cmd,optional"`
}

type Remote struct {
	Port string `hcl:"port"`
	Host string `hcl:"host"`
}

func Parse(configfile string) ([]Tunnel, error) {
	parser := hclparse.NewParser()
	file, diags := parser.ParseHCLFile(configfile)

	if diags.HasErrors() {
		return nil, fmt.Errorf("parse config: %w", diags)
	}

	var config Configuration
	confDiags := gohcl.DecodeBody(file.Body, nil, &config)

	if confDiags.HasErrors() {
		return nil, fmt.Errorf("decode config: %w", confDiags)
	}

	printConfig(config.Tunnels)

	return config.Tunnels, nil
}

func printConfig(config []Tunnel) {
	w := tabwriter.NewWriter(os.Stdout, 1, 1, 1, ' ', tabwriter.TabIndent)

	blue := color.New(color.FgHiBlue)
	green := color.New(color.FgGreen)

	_, _ = fmt.Fprintf(w, "%s\t%s\t\t%s\n", "User@Host", blue.Sprintf("Remote"), green.Sprintf("Local"))

	for _, tunnel := range config {

		from := blue.Sprintf("127.0.0.1:%s", tunnel.Local.Port)
		if tunnel.Name != "" {
			from = blue.Sprintf("%s", tunnel.Name)
		}

		to := green.Sprintf("127.0.0.1:%s", tunnel.Remote.Port)
		_, _ = fmt.Fprintf(w, "%s@%s\t%s\t->\t%s\n", tunnel.User, tunnel.Remote.Host, from, to)
	}
	_ = w.Flush()
}

func FindConfig(debug *bool) (string, error) {
	green := color.New(color.FgGreen)
	homedir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home directory: %w", err)
	}
	currentDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}
	searchPaths := []string{
		currentDir + "/ssh-tunnel.hcl",
		homedir + "/.config/ssh-tunnel/ssh-tunnel.hcl",
		"/etc/ssh-tunnel/ssh-tunnel.hcl",
	}

	for _, spath := range searchPaths {
		if *debug {
			fmt.Println("Searching for config:", spath)
		}
		if _, err := os.Stat(spath); err == nil {
			if *debug {
				_, _ = green.Println("Found config in", spath)
			}
			return spath, nil
		}
	}

	return "", errors.New("no ssh-tunnel.hcl found")
}
