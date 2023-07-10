package config

import (
	"fmt"
	"log"
	"os"
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

// String prints out a pretty version of the tunnel struct
func (t Tunnel) String() {

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
