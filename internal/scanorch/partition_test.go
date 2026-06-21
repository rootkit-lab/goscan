package scanorch

import "testing"

func TestPartitionRoundRobin(t *testing.T) {
	domains := []string{"a", "b", "c", "d", "e"}
	chunks := Partition(domains, 3)
	if len(chunks) != 3 {
		t.Fatalf("expected 3 chunks, got %d", len(chunks))
	}
	if len(chunks[0]) != 2 || len(chunks[1]) != 2 || len(chunks[2]) != 1 {
		t.Fatalf("unexpected sizes: %v", []int{len(chunks[0]), len(chunks[1]), len(chunks[2])})
	}
}
