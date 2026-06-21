package remoteworker

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"golang.org/x/crypto/ssh"
)

// TestConnection verifies SSH credentials and returns remote goscan version if installed.
func TestConnection(w Config) (remoteVersion string, err error) {
	client, err := dial(w)
	if err != nil {
		return "", err
	}
	defer client.Close()

	home, err := remoteHome(client)
	if err != nil {
		return "", err
	}
	bin := remoteAppBin(home)
	const marker = "GOSCAN_SSH_OK"
	script := strings.TrimSpace(fmt.Sprintf(`
printf '%%s\n' %s
if test -x %s; then
  %s --version 2>/dev/null | head -1 || printf 'none\n'
else
  printf 'none\n'
fi`, shellQuote(marker), shellQuote(bin), shellQuote(bin)))
	out, err := runSession(client, "exec sh -c "+shellQuote(script))
	if err != nil {
		return "", err
	}
	lines := remoteLines(out)
	ver, ok := lineAfterMarker(lines, marker)
	if !ok {
		snip := strings.TrimSpace(out)
		if len(snip) > 160 {
			snip = snip[:160] + "…"
		}
		return "", fmt.Errorf("resposta SSH inesperada (%s)", snip)
	}
	if ver != "" && ver != "none" {
		remoteVersion = ver
	}
	return remoteVersion, nil
}

func remoteLines(out string) []string {
	out = strings.TrimSpace(out)
	if out == "" {
		return nil
	}
	lines := make([]string, 0, strings.Count(out, "\n")+1)
	for _, l := range strings.Split(out, "\n") {
		l = strings.TrimSpace(strings.TrimSuffix(l, "\r"))
		if l != "" {
			lines = append(lines, l)
		}
	}
	return lines
}

func lineAfterMarker(lines []string, marker string) (string, bool) {
	for i, l := range lines {
		if l == marker && i+1 < len(lines) {
			return lines[i+1], true
		}
	}
	return "", false
}

func runSession(client *ssh.Client, cmd string) (string, error) {
	return runSessionCtx(context.Background(), client, cmd)
}

func runSessionCtx(ctx context.Context, client *ssh.Client, cmd string) (string, error) {
	return runSessionCtxIO(ctx, client, cmd, false)
}

func runSessionStdoutCtx(ctx context.Context, client *ssh.Client, cmd string) (string, error) {
	return runSessionCtxIO(ctx, client, cmd, true)
}

func runSessionCtxIO(ctx context.Context, client *ssh.Client, cmd string, stdoutOnly bool) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()
	var outBuf, errBuf bytes.Buffer
	session.Stdout = &outBuf
	if stdoutOnly {
		session.Stderr = &errBuf
	} else {
		session.Stderr = &outBuf
	}
	if err := session.Start(cmd); err != nil {
		return "", err
	}
	waitDone := make(chan error, 1)
	go func() { waitDone <- session.Wait() }()
	select {
	case <-ctx.Done():
		_ = session.Close()
		return outBuf.String(), ctx.Err()
	case err := <-waitDone:
		combined := outBuf.String()
		if err != nil {
			detail := strings.TrimSpace(combined)
			if detail == "" {
				detail = strings.TrimSpace(errBuf.String())
			}
			if detail != "" {
				return combined, fmt.Errorf("%w: %s", err, detail)
			}
			return combined, err
		}
		return combined, nil
	}
}

func runSessionCombined(client *ssh.Client, cmd string) (stdout, stderr string, err error) {
	session, err := client.NewSession()
	if err != nil {
		return "", "", err
	}
	defer session.Close()
	var out, errBuf bytes.Buffer
	session.Stdout = &out
	session.Stderr = &errBuf
	err = session.Run(cmd)
	return out.String(), errBuf.String(), err
}
