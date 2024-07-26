package tunnel

import (
	"io"
	"log"
	"net"
	"os"

	"github.com/inveracity/ssh-tunnel/internal/config"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

func Start(config config.Tunnel) {
	sock, err := sshAgent()
	if err != nil {
		log.Fatalln(err)
	}

	cfg, err := makeSshConfig(config.User, sock)
	if err != nil {
		log.Fatalln(err)
	}

	conn, err := ssh.Dial("tcp", config.Remote.Host, cfg)
	if err != nil {
		log.Fatalln(err)
	}

	defer conn.Close()

	if err := tunnel(conn, "localhost:"+config.Local.Port, "localhost:"+config.Remote.Port); err != nil {
		log.Fatalf("failed to tunnel traffic: %q", err)
	}
}

func sshAgent() (agent.ExtendedAgent, error) {
	sshAgent, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK"))
	if err != nil {
		return nil, err
	}

	return agent.NewClient(sshAgent), nil
}

func makeSshConfig(user string, sock agent.ExtendedAgent) (*ssh.ClientConfig, error) {
	signers, err := sock.Signers()
	if err != nil {
		log.Printf("create signers error: %s", err)
		return nil, err
	}

	config := ssh.ClientConfig{
		User:            user,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signers...),
		},
	}

	return &config, nil
}

func tunnel(conn *ssh.Client, local, remote string) error {
	pipe := func(writer, reader net.Conn) {
		defer writer.Close()
		defer reader.Close()

		_, err := io.Copy(writer, reader)
		if err != nil {
			log.Printf("failed to copy: %s", err)
		}
	}

	listener, err := net.Listen("tcp", local)
	if err != nil {
		return err
	}

	for {
		here, err := listener.Accept()
		if err != nil {
			return err
		}

		go func(here net.Conn) {
			there, err := conn.Dial("tcp", remote)
			if err != nil {
				log.Fatalf("failed to dial to remote: %q", err)
			} else {
				go pipe(there, here)
				go pipe(here, there)
			}
		}(here)
	}
}
