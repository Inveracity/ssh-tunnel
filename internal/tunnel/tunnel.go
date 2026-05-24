package tunnel

import (
	"io"
	"log"
	"net"
	"os"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

type Tunnel struct {
	User   string
	Local  Local
	Remote Remote
}

type Local struct {
	Port string
}

type Remote struct {
	Port string
	Host string
}

func Start(t Tunnel) error {
	sock, err := sshAgent()
	if err != nil {
		return err
	}

	cfg, err := makeSshConfig(t.User, sock)
	if err != nil {
		return err
	}

	conn, err := ssh.Dial("tcp", t.Remote.Host, cfg)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()

	if err := tunnel(conn, "localhost:"+t.Local.Port, "localhost:"+t.Remote.Port); err != nil {
		return err
	}

	return nil
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

func pipe(writer, reader net.Conn) {
	defer func() { _ = writer.Close() }()
	defer func() { _ = reader.Close() }()

	_, err := io.Copy(writer, reader)
	if err != nil {
		log.Printf("failed to copy: %s", err)
	}
}

func tunnel(conn *ssh.Client, local, remote string) error {
	listener, err := net.Listen("tcp", local)
	if err != nil {
		return err
	}

	for {
		here, err := listener.Accept()
		if err != nil {
			return err
		}

		there, err := conn.Dial("tcp", remote)
		if err != nil {
			log.Printf("failed to dial to remote: %q", err)
			_ = here.Close()
			continue
		}

		go pipe(there, here)
		go pipe(here, there)
	}
}
