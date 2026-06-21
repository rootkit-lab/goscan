//go:build !nosqlite

package scanner

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"strings"

	"goscan/internal/store"
)

const ImportBatchSize = 5000

// ImportProgress reports bulk domain import progress.
type ImportProgress func(imported int64, fileLabel string)

// ImportDomainsFromFilesCtx ingests domain lists in optimized batches.
func ImportDomainsFromFilesCtx(ctx context.Context, files []string, ds *store.DomainStore, onProgress ImportProgress) (int64, error) {
	var total int64
	for _, path := range files {
		if ctx.Err() != nil {
			return total, ctx.Err()
		}
		label := filepath.Base(path)
		var n int64
		var err error
		if isDomainListFile(path) {
			n, err = importDomainListBatch(ctx, path, ds, onProgress, &total, label)
		} else if isEnvFile(path) {
			n, err = importEnvFileBatch(ctx, path, ds, onProgress, &total, label)
		}
		if err != nil {
			return total, err
		}
		if n > 0 && onProgress != nil {
			onProgress(total, label)
		}
	}
	if err := ds.Flush(); err != nil {
		return total, err
	}
	return total, nil
}

func importDomainListBatch(ctx context.Context, path string, ds *store.DomainStore, onProgress ImportProgress, total *int64, label string) (int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	buf := make([]byte, 256*1024)
	sc.Buffer(buf, 4*1024*1024)

	batch := make([]string, 0, ImportBatchSize)
	var fileAdded int64
	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		added, err := ds.ImportBatch(batch, path)
		if err != nil {
			return err
		}
		fileAdded += int64(added)
		*total += int64(added)
		batch = batch[:0]
		if added > 0 && onProgress != nil {
			onProgress(*total, label)
		}
		return nil
	}

	for sc.Scan() {
		if ctx.Err() != nil {
			return fileAdded, ctx.Err()
		}
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		d := cleanDomain(line)
		if !isValidDomain(d) {
			continue
		}
		batch = append(batch, d)
		if len(batch) >= ImportBatchSize {
			if err := flush(); err != nil {
				return fileAdded, err
			}
		}
	}
	if err := sc.Err(); err != nil {
		return fileAdded, err
	}
	return fileAdded, flush()
}

func importEnvFileBatch(ctx context.Context, path string, ds *store.DomainStore, onProgress ImportProgress, total *int64, label string) (int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	buf := make([]byte, 256*1024)
	sc.Buffer(buf, 4*1024*1024)

	batch := make([]string, 0, ImportBatchSize)
	seen := make(map[string]struct{}, ImportBatchSize)
	var fileAdded int64

	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		added, err := ds.ImportBatch(batch, path)
		if err != nil {
			return err
		}
		fileAdded += int64(added)
		*total += int64(added)
		batch = batch[:0]
		if added > 0 && onProgress != nil {
			onProgress(*total, label)
		}
		return nil
	}

	add := func(d string) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if !isValidDomain(d) {
			return nil
		}
		if _, ok := seen[d]; ok {
			return nil
		}
		seen[d] = struct{}{}
		batch = append(batch, d)
		if len(batch) >= ImportBatchSize {
			return flush()
		}
		return nil
	}

	for sc.Scan() {
		if ctx.Err() != nil {
			return fileAdded, ctx.Err()
		}
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		for _, url := range reURL.FindAllString(line, -1) {
			if domain := extractDomain(url); domain != "" {
				if err := add(domain); err != nil {
					return fileAdded, err
				}
			}
		}
		for _, d := range reDomain.FindAllString(line, -1) {
			if err := add(cleanDomain(d)); err != nil {
				return fileAdded, err
			}
		}
		for _, match := range reEmail.FindAllStringSubmatch(line, -1) {
			if len(match) > 1 {
				if err := add(cleanDomain(match[1])); err != nil {
					return fileAdded, err
				}
			}
		}
	}
	if err := sc.Err(); err != nil {
		return fileAdded, err
	}
	return fileAdded, flush()
}
