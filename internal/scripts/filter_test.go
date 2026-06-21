package scripts

import "testing"

func TestResolveBatchFilter(t *testing.T) {
	tests := []struct {
		in   string
		want []string
	}{
		{"mysql", []string{"chk-mysql"}},
		{"MySQL", []string{"chk-mysql"}},
		{"db", []string{"chk-mysql", "chk-postgres", "chk-mongodb"}},
		{"email", []string{"chk-smtp", "chk-sendgrid", "chk-mailgun"}},
		{"chk-redis", []string{"chk-redis"}},
	}
	for _, tc := range tests {
		got, err := ResolveBatchFilter(tc.in)
		if err != nil {
			t.Fatalf("%q: %v", tc.in, err)
		}
		if len(got) != len(tc.want) {
			t.Fatalf("%q: got %v want %v", tc.in, got, tc.want)
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Fatalf("%q: got %v want %v", tc.in, got, tc.want)
			}
		}
	}
	if _, err := ResolveBatchFilter("nope"); err == nil {
		t.Fatal("expected error for unknown filter")
	}
}

func TestScriptAllowed(t *testing.T) {
	opts := BatchPlanOpts{ScriptIDs: []string{"chk-mysql", "chk-postgres"}}
	if !scriptAllowed("chk-mysql", opts) {
		t.Fatal("mysql should be allowed")
	}
	if scriptAllowed("chk-redis", opts) {
		t.Fatal("redis should not be allowed")
	}
	single := BatchPlanOpts{ScriptID: "chk-smtp"}
	if !scriptAllowed("chk-smtp", single) {
		t.Fatal("smtp should be allowed")
	}
}
