package remoteworker

import (
	"bufio"
	"context"
	"io"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/crypto/ssh"
)

func runSessionStream(ctx context.Context, client *ssh.Client, cmd string, onLine func(string)) error {
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	stdout, err := session.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := session.StderrPipe()
	if err != nil {
		return err
	}
	if err := session.Start(cmd); err != nil {
		return err
	}

	var wg sync.WaitGroup
	pump := func(r io.Reader) {
		defer wg.Done()
		sc := bufio.NewScanner(r)
		buf := make([]byte, 64*1024)
		sc.Buffer(buf, 1024*1024)
		for sc.Scan() {
			if ctx.Err() != nil {
				return
			}
			if onLine != nil {
				onLine(sc.Text())
			}
		}
	}
	wg.Add(2)
	go pump(stdout)
	go pump(stderr)

	waitDone := make(chan error, 1)
	go func() { waitDone <- session.Wait() }()

	select {
	case <-ctx.Done():
		_ = session.Close()
		wg.Wait()
		return ctx.Err()
	case err := <-waitDone:
		wg.Wait()
		return err
	}
}

func parseProgressLine(line string) (scanned, vulns, total int64, ok bool) {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "@goscan/progress ") {
		return 0, 0, 0, false
	}
	for _, part := range strings.Fields(line) {
		if part == "@goscan/progress" {
			continue
		}
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		n, err := strconv.ParseInt(kv[1], 10, 64)
		if err != nil {
			continue
		}
		switch kv[0] {
		case "scanned":
			scanned = n
		case "vulns":
			vulns = n
		case "total":
			total = n
		}
	}
	return scanned, vulns, total, total > 0 || scanned > 0
}

func parseFoundLine(line string) (domain, path, url string, ok bool) {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "@goscan/found ") {
		return "", "", "", false
	}
	for _, part := range strings.Fields(line) {
		if part == "@goscan/found" {
			continue
		}
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case "domain":
			domain = kv[1]
		case "path":
			path = kv[1]
		case "url":
			url = kv[1]
		}
	}
	return domain, path, url, domain != "" && path != ""
}
