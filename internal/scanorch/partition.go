package scanorch

import (
	"bufio"
	"os"
	"path/filepath"
)

// Partition splits domains round-robin into n chunks.
func Partition(domains []string, n int) [][]string {
	if n <= 0 {
		return nil
	}
	chunks := make([][]string, n)
	for i, d := range domains {
		chunks[i%n] = append(chunks[i%n], d)
	}
	return chunks
}

// WriteChunkDir writes domains to chunk/domains.txt and returns cleanup.
func WriteChunkDir(domains []string) (dir string, cleanup func(), err error) {
	dir, err = os.MkdirTemp("", "goscan-chunk-*")
	if err != nil {
		return "", nil, err
	}
	cleanup = func() { _ = os.RemoveAll(dir) }
	path := filepath.Join(dir, "domains.txt")
	f, err := os.Create(path)
	if err != nil {
		cleanup()
		return "", nil, err
	}
	w := bufio.NewWriter(f)
	for _, d := range domains {
		if _, err := w.WriteString(d + "\n"); err != nil {
			f.Close()
			cleanup()
			return "", nil, err
		}
	}
	if err := w.Flush(); err != nil {
		f.Close()
		cleanup()
		return "", nil, err
	}
	if err := f.Close(); err != nil {
		cleanup()
		return "", nil, err
	}
	return dir, cleanup, nil
}
