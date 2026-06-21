package main

import (
	"fmt"
	"os"

	"goscan/internal/batchlog"
)

func runBatchAnalyzeCLI() {
	_, dataRoot := resolveRoots()
	args := os.Args[1:]
	if len(args) == 0 {
		args = []string{"--last"}
	}
	runDir, err := batchlog.ResolveRunDir(dataRoot, args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		os.Exit(1)
	}
	if err := batchlog.AnalyzeRun(runDir); err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		os.Exit(1)
	}
}
