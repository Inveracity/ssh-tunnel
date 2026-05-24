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
	tunnel   Tunnel
	config   Config
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	mu       sync.Mutex
	listener net.Listener
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

// Close cancels the runner's context, closes the listener, and waits for all active pipe goroutines to finish.
// Safe to call multiple times.
func (r *TunnelRunner) Close() {
	r.mu.Lock()
	r.cancel()
	if r.listener != nil {
		_ = r.listener.Close()
	}
	r.mu.Unlock()
	r.wg.Wait()
}

// Run establishes the SSH connection and starts the accept loop.
// It blocks until the context is cancelled or a permanent error occurs.
func (r *TunnelRunner) Run() error {
	sock, err := sshAgent()
	if err != nil {
		return err
	}

	sshCfg, err := makeSshConfig(r.tunnel.User, sock)
	if err != nil {
		return err
	}

	conn, err := ssh.Dial("tcp", r.tunnel.Remote.Host, sshCfg)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()

	local := "localhost:" + r.tunnel.Local.Port
	remote := "localhost:" + r.tunnel.Remote.Port

	return r.runAcceptLoop(conn, local, remote)
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

// acceptConnection accepts a single connection from the listener, returning early if context is cancelled.
func (r *TunnelRunner) acceptConnection(listener net.Listener) (net.Conn, error) {
	done := make(chan struct{})
	var conn net.Conn
	var err error

	go func() {
		conn, err = listener.Accept()
		close(done)
	}()

	select {
	case <-r.ctx.Done():
		_ = listener.Close()
		<-done
		return nil, r.ctx.Err()
	case <-done:
		return conn, err
	}
}

// dialRemote dials the remote endpoint through the SSH connection.
func dialRemote(conn *ssh.Client, remote string) (net.Conn, error) {
	return conn.Dial("tcp", remote)
}

// runAcceptLoop accepts connections and spawns pipe pairs until the context is cancelled or a permanent error occurs.
func (r *TunnelRunner) runAcceptLoop(conn *ssh.Client, local, remote string) error {
	listener, err := net.Listen("tcp", local)
	if err != nil {
		return err
	}

	r.mu.Lock()
	r.listener = listener
	r.mu.Unlock()

	defer func() {
		r.mu.Lock()
		r.listener = nil
		r.mu.Unlock()
		_ = listener.Close()
	}()

	for {
		here, err := r.acceptConnection(listener)
		if err != nil {
			if r.ctx.Err() != nil {
				return nil
			}
			return err
		}

		there, err := dialRemote(conn, remote)
		if err != nil {
			log.Printf("failed to dial to remote: %q", err)
			_ = here.Close()
			continue
		}

		r.spawnPipePair(here, there)
	}
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
