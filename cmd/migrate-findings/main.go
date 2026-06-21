package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"goscan/internal/paths"
	"goscan/internal/store"
)

func main() {
	appRoot, err := paths.AppRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "app root: %v\n", err)
		os.Exit(1)
	}
	dataRoot, err := paths.DataRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "data root: %v\n", err)
		os.Exit(1)
	}

	dbPath := paths.DefaultDBPath(dataRoot)
	findingsDir := paths.FindingsRoot(dataRoot)
	archiveDir := paths.ArchiveDir(dataRoot)

	domainStore, err := store.OpenDomainStore(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "db: %v\n", err)
		os.Exit(1)
	}
	defer domainStore.Close()

	fs, err := store.OpenFindingsStore(domainStore.DB(), findingsDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "findings: %v\n", err)
		os.Exit(1)
	}

	var moved, dupes, errors int
	matches, _ := filepath.Glob(filepath.Join(appRoot, "scan_resultados_*"))
	for _, scanDir := range matches {
		info, err := os.Stat(scanDir)
		if err != nil || !info.IsDir() {
			continue
		}
		runID := strings.TrimPrefix(filepath.Base(scanDir), "scan_resultados_")
		entries, err := os.ReadDir(scanDir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".env") {
				continue
			}
			content, err := os.ReadFile(filepath.Join(scanDir, e.Name()))
			if err != nil {
				errors++
				continue
			}
			ok, err := fs.ImportLegacyFile(runID, e.Name(), content, "", "HIGH", true)
			if err != nil {
				fmt.Fprintf(os.Stderr, "erro %s: %v\n", e.Name(), err)
				errors++
				continue
			}
			if ok {
				moved++
			} else {
				dupes++
			}
		}

		dest := filepath.Join(archiveDir, filepath.Base(scanDir))
		_ = os.MkdirAll(archiveDir, 0755)
		if _, err := os.Stat(dest); os.IsNotExist(err) {
			if err := os.Rename(scanDir, dest); err != nil {
				fmt.Fprintf(os.Stderr, "arquivar %s: %v\n", scanDir, err)
			}
		}
	}

	count, _ := fs.Count()
	fmt.Printf("Migracao concluida: importados=%d duplicados=%d erros=%d total_findings=%d\n", moved, dupes, errors, count)
}
