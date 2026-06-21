package remoteworker

import (
	"context"
	"io"
	"net"
	"sync"

	"golang.org/x/crypto/ssh"
)

// Tunnel proxies remote TCP connections to a local address via SSH remote forward.
type Tunnel struct {
	ln       net.Listener
	local    string
	remote   string
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

// OpenTunnel listens on remoteAddr and forwards each connection to localAddr.
func OpenTunnel(ctx context.Context, client *ssh.Client, remoteAddr, localAddr string) (*Tunnel, error) {
	ln, err := client.Listen("tcp", remoteAddr)
	if err != nil {
		return nil, err
	}
	tctx, cancel := context.WithCancel(ctx)
	t := &Tunnel{ln: ln, local: localAddr, remote: remoteAddr, cancel: cancel}
	t.wg.Add(1)
	go t.run(tctx)
	return t, nil
}

func (t *Tunnel) RemoteAddr() string { return t.remote }

func (t *Tunnel) run(ctx context.Context) {
	defer t.wg.Done()
	for {
		if ctx.Err() != nil {
			return
		}
		rc, err := t.ln.Accept()
		if err != nil {
			return
		}
		go t.handle(ctx, rc)
	}
}

func (t *Tunnel) handle(ctx context.Context, remote net.Conn) {
	defer remote.Close()
	lc, err := net.Dial("tcp", t.local)
	if err != nil {
		return
	}
	defer lc.Close()
	done := make(chan struct{}, 2)
	go func() { _, _ = io.Copy(lc, remote); done <- struct{}{} }()
	go func() { _, _ = io.Copy(remote, lc); done <- struct{}{} }()
	select {
	case <-ctx.Done():
	case <-done:
		<-done
	}
}

func (t *Tunnel) Close() {
	t.cancel()
	_ = t.ln.Close()
	t.wg.Wait()
}
