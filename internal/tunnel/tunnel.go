package tunnel

import (
	"context"
	"io"
	"log"
	"net"
	"os"
	"sync"
	"time"

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
	Cmd  []string
}

type Remote struct {
	Port string
	Host string
}

// Config holds tunable parameters for TunnelRunner.
type Config struct {
	MaxRetries       int
	RetryBaseDelay   time.Duration
	RetryMaxDelay    time.Duration
	AcceptRetryDelay time.Duration
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		MaxRetries:       0, // unlimited
		RetryBaseDelay:   time.Second,
		RetryMaxDelay:    30 * time.Second,
		AcceptRetryDelay: 100 * time.Millisecond,
	}
}

// TunnelRunner manages the lifecycle of a single SSH tunnel.
type TunnelRunner struct {
	tunnel Tunnel
	config Config
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	mu     sync.Mutex
}

// NewTunnelRunner creates a TunnelRunner with a cancellable child context.
func NewTunnelRunner(ctx context.Context, t Tunnel, cfg Config) *TunnelRunner {
	childCtx, cancel := context.WithCancel(ctx)
	return &TunnelRunner{
		tunnel: t,
		config: cfg,
		ctx:    childCtx,
		cancel: cancel,
	}
}

// Close cancels the runner's context and waits for all active pipe goroutines to finish.
// Safe to call multiple times.
func (r *TunnelRunner) Close() {
	r.mu.Lock()
	r.cancel()
	r.mu.Unlock()
	r.wg.Wait()
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

	if err := tunnel(context.Background(), conn, "localhost:"+t.Local.Port, "localhost:"+t.Remote.Port); err != nil {
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

func pipe(ctx context.Context, writer, reader net.Conn, closeOnce *sync.Once) {
	go func() {
		<-ctx.Done()
		_ = reader.Close()
	}()

	_, err := io.Copy(writer, reader)
	closeOnce.Do(func() {
		_ = writer.Close()
		_ = reader.Close()
	})
	if err != nil {
		log.Printf("failed to copy: %s", err)
	}
}

// spawnPipePair creates a bidirectional pipe between local and remote connections.
// Both directions share a sync.Once to ensure connections are closed exactly once.
//
//nolint:unused // will be wired into runAcceptLoop in Step 3
func (r *TunnelRunner) spawnPipePair(local, remote net.Conn) {
	var closeOnce sync.Once
	r.wg.Add(2)

	go func() {
		defer r.wg.Done()
		pipe(r.ctx, remote, local, &closeOnce)
	}()

	go func() {
		defer r.wg.Done()
		pipe(r.ctx, local, remote, &closeOnce)
	}()
}

func tunnel(ctx context.Context, conn *ssh.Client, local, remote string) error {
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

		var closeOnce sync.Once
		go pipe(ctx, there, here, &closeOnce)
		go pipe(ctx, here, there, &closeOnce)
	}
}
