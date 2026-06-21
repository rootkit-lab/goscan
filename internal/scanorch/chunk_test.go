package scanorch

import "testing"

func TestWorkerChunkSize(t *testing.T) {
	if got := workerChunkSize(2); got != 2500 {
		t.Fatalf("workerChunkSize(2) = %d, want 2500", got)
	}
	if got := workerChunkSize(10); got != 500 {
		t.Fatalf("workerChunkSize(10) = %d, want 500 (min)", got)
	}
}
