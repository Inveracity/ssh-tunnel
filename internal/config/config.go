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
	User   string `hcl:"user"`
	Local  Local  `hcl:"local,block"`
	Remote Remote `hcl:"remote,block"`
}

type Local struct {
	Port string `hcl:"port"`
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

	w := tabwriter.NewWriter(os.Stdout, 6, 1, 2, ' ', tabwriter.TabIndent)
	fmt.Fprintln(w, "Remote Host\tTunnel")
	for _, tunnel := range config.Tunnels {
		greentunnel := color.New(color.FgGreen).Sprintf("127.0.0.1:%s -> 127.0.0.1:%s", tunnel.Local.Port, tunnel.Remote.Port)
		fmt.Fprintf(w, "%s@%s\t%s\n", tunnel.User, tunnel.Remote.Host, greentunnel)
	}
	w.Flush()

	return config.Tunnels
}
