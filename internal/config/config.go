package config

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/fatih/color"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclparse"
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

func Parse(configfile string) []Tunnel {
	parser := hclparse.NewParser()
	file, diags := parser.ParseHCLFile(configfile)

	if diags.HasErrors() {
		log.Fatal(diags)
	}

	var config Configuration
	confDiags := gohcl.DecodeBody(file.Body, nil, &config)

	if confDiags.HasErrors() {
		log.Fatal(confDiags)
	}

	printConfig(config.Tunnels)

	return config.Tunnels
}

func printConfig(config []Tunnel) {
	w := tabwriter.NewWriter(os.Stdout, 1, 1, 1, ' ', tabwriter.TabIndent)

	blue := color.New(color.FgHiBlue)
	green := color.New(color.FgGreen)

	fmt.Fprintf(w, "%s\t%s\t\t%s\n", "User@Host", blue.Sprintf("Remote"), green.Sprintf("Local"))

	for _, tunnel := range config {

		from := blue.Sprintf("127.0.0.1:%s", tunnel.Local.Port)
		if tunnel.Name != "" {
			from = blue.Sprintf("%s", tunnel.Name)
		}

		to := green.Sprintf("127.0.0.1:%s", tunnel.Remote.Port)
		fmt.Fprintf(w, "%s@%s\t%s\t->\t%s\n", tunnel.User, tunnel.Remote.Host, from, to)
	}
	w.Flush()
}

func FindConfig(debug *bool) (string, error) {
	green := color.New(color.FgGreen)
	homedir, _ := os.UserHomeDir()
	currentDir := cwd()
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
				green.Println("Found config in", spath)
			}
			return spath, nil
		}
	}

	return "", errors.New("no ssh-tunnel.hcl found")
}

func cwd() string {
	ex, err := filepath.Abs("./")
	if err != nil {
		panic(err)
	}
	return ex
}
