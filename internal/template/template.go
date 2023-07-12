package template

import (
	"log"
	"os"
)

var filename = "ssh-tunnel.hcl"

func Write() {

	perms := os.FileMode(0644)

	if !exists() {
		os.WriteFile(filename, []byte(template()), perms)
	} else {
		log.Println("a ssh-tunnel-hcl file already exists, nothing to do")
	}
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
