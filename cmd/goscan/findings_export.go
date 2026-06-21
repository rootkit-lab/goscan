package main

import (
	"flag"
	"fmt"
	"os"

	"goscan/internal/paths"
	"goscan/internal/store"
)

func runFindingsExportJSON() {
	_, dataRoot := resolveRoots()
	dbPath := flag.String("db", paths.DefaultDBPath(dataRoot), "SQLite de domínios")
	findingsDir := flag.String("findings", paths.FindingsRoot(dataRoot), "Diretório de findings")
	runID := flag.String("run-id", "", "Filtrar por scan run id")
	flag.Parse()

	domainStore, err := store.OpenDomainStore(*dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		os.Exit(1)
	}
	defer domainStore.Close()

	fs, err := store.OpenFindingsStore(domainStore.DB(), *findingsDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		os.Exit(1)
	}
	if err := fs.ExportFindingsJSON(os.Stdout, *runID); err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		os.Exit(1)
	}
}
