//go:build !nosqlite

package scanner

import (
	"context"

	"goscan/internal/store"
)

// FindInputFiles lists .env and .txt inputs under rootDir.
func FindInputFiles(rootDir string) []string {
	return findInputFiles(rootDir)
}

// CollectPendingDomains ingests files and returns domains queued for scan.
func CollectPendingDomains(files []string, ds *store.DomainStore, rescan bool) []string {
	return CollectPendingDomainsCtx(context.Background(), files, ds, rescan)
}

// CollectPendingDomainsCtx ingests files using batch import (rescan ignored — only new domains are inserted).
func CollectPendingDomainsCtx(ctx context.Context, files []string, ds *store.DomainStore, rescan bool) []string {
	_, _ = ImportDomainsFromFilesCtx(ctx, files, ds, nil)
	return nil
}
