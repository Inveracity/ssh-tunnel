package main

import (
	"flag"
	"fmt"
	"os/exec"
	"sync"

	"github.com/fatih/color"
	"github.com/inveracity/ssh-tunnel/internal/config"
	"github.com/inveracity/ssh-tunnel/internal/template"
	"github.com/inveracity/ssh-tunnel/internal/tunnel"
	"github.com/inveracity/ssh-tunnel/internal/version"
)

var (
	configfile  = flag.String("config", "", "The config file to use")
	flagNoColor = flag.Bool("no-color", false, "Disable color output")
	v           = flag.Bool("version", false, "Print the version and exit")
	newconfig   = flag.Bool("init", false, "Create a new config file")
	debug       = flag.Bool("debug", false, "Print debug output")
)

func main() {
	flag.Parse()

	if *flagNoColor {
		color.NoColor = true
	}

	if *v {
		PrintVersion()
		return
	}

	if *newconfig {
		if err := template.Write(); err != nil {
			fmt.Println("ERROR:", err)
		}
		return
	}

	if err := run(debug); err != nil {
		fmt.Println("ERROR:", err)
	}
}

func run(debug *bool) (err error) {
	var path_to_config string
	if *configfile == "" {
		path_to_config, err = config.FindConfig(debug)
		if err != nil {
			return err
		}
	} else {
		path_to_config = *configfile
	}

	tunnels, err := config.Parse(path_to_config)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	for _, t := range tunnels {
		wg.Add(1)
		go func(t config.Tunnel) {
			defer wg.Done()
			tunnel.Start(t)
		}(t)

		if t.Local.Cmd != nil {
			command, args := t.Local.Cmd[0], t.Local.Cmd[1:]
			if err := exec.Command(command, args...).Run(); err != nil {
				fmt.Printf("WARN: failed to run post-tunnel command %q: %v\n", command, err)
			}
		}
	}

	wg.Wait()
	return nil
}

func PrintVersion() {
	fmt.Println(version.Version)
}
