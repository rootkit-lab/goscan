package store

import "testing"

func TestParseLegacyFilename(t *testing.T) {
	tests := []struct {
		name   string
		file   string
		domain string
		path   string
		ok     bool
	}{
		{"env root", "colore_shampora_it__env.env", "colore.shampora.it", "/.env", true},
		{"env local", "citoyens_vitrolles13_fr__env_local.env", "citoyens.vitrolles13.fr", "/.env.local", true},
		{"bad", "invalid.env", "", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, p, ok := ParseLegacyFilename(tt.file)
			if ok != tt.ok {
				t.Fatalf("ok=%v want %v", ok, tt.ok)
			}
			if d != tt.domain || p != tt.path {
				t.Fatalf("got %q %q want %q %q", d, p, tt.domain, tt.path)
			}
		})
	}
}
