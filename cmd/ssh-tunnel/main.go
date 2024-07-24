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
		template.Write()
		return
	}

	if err := run(); err != nil {
		fmt.Println("ERROR:", err)
	}
}

func run() (err error) {
	var path_to_config string
	if *configfile == "" {
		path_to_config, err = config.FindConfig()
		if err != nil {
			return err
		}
	} else {
		path_to_config = *configfile
	}

	config := config.Parse(path_to_config)

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
	return nil
}

func PrintVersion() {
	fmt.Println(version.Version)
}
