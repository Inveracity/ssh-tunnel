package main

import (
	"flag"
	"fmt"
	"os/exec"
	"sync"

	"github.com/fatih/color"
	"github.com/inveracity/ssh-tunnel/internal/config"
	"github.com/inveracity/ssh-tunnel/internal/tunnel"
	"github.com/inveracity/ssh-tunnel/internal/version"
)

var (
	configfile  = flag.String("config", "config.hcl", "The config file to use")
	flagNoColor = flag.Bool("no-color", false, "Disable color output")
	v           = flag.Bool("version", false, "Print the version and exit")
)

func main() {
	flag.Parse()

	if *flagNoColor {
		color.NoColor = true // disables colorized output
	}

	if *v {
		PrintVersion()
		return
	}

	config := config.Parse(*configfile)

	var wg sync.WaitGroup
	for _, c := range config {
		wg.Add(1)
		defer wg.Done()
		go tunnel.Start(c)

		// If there is a command to run, run it
		if c.Local.Cmd != nil {
			command, args := c.Local.Cmd[0], c.Local.Cmd[1:]
			exec.Command(command, args...).Run()
		}
	}

	wg.Wait()
}

func PrintVersion() {
	fmt.Println(version.Version)
}
