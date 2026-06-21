package main

import (
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"os"

	"goscan/internal/workerapi"
)

func runWorkerCLI() {
	listen := flag.String("listen", ":9090", "Endereço HTTP")
	token := flag.String("token", "", "Token Bearer (gerado se vazio)")
	flag.Parse()

	tok := *token
	if tok == "" {
		b := make([]byte, 16)
		_, _ = rand.Read(b)
		tok = hex.EncodeToString(b)
		fmt.Fprintf(os.Stderr, "worker token: %s\n", tok)
	}

	srv, err := workerapi.NewFromRoots(tok)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "🌐 worker API em %s\n", *listen)
	if err := srv.ListenAndServe(*listen); err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		os.Exit(1)
	}
}
