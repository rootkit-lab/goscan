package main

import "goscan/internal/checker"

func summarizeOutput(s string) string {
	return checker.SummarizeOutput(s)
}

func classifyCheckerStatus(exitCode int, output string) string {
	return checker.ClassifyStatus(exitCode, output)
}
