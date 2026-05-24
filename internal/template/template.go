package template

import (
	"fmt"
	"log"
	"os"
)

var filename = "ssh-tunnel.hcl"

func Write() error {
	perms := os.FileMode(0644)

	if !exists() {
		if err := os.WriteFile(filename, []byte(template()), perms); err != nil {
			return fmt.Errorf("failed to write %s: %w", filename, err)
		}
	} else {
		log.Println("a ssh-tunnel.hcl file already exists, nothing to do")
	}

	return nil
}

func template() string {
	return `
tunnel {
	local {
		port = 8080
	}
	remote {
		host = "1.2.3.4:22"
		port = 8080
	}
}
`
}

func exists() bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}
