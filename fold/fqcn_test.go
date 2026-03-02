package fold

import (
	"testing"
)

func TestResolveFQCN(t *testing.T) {
	reg := DefaultRegistry()

	tests := []struct {
		fqcn string
		want string
		err  bool
	}{
		{"origami.marbles.rca", "github.com/dpopsuev/origami/marbles/rca", false},
		{"origami.adapters.rp", "github.com/dpopsuev/origami/adapters/rp", false},
		{"origami.adapters.sqlite", "github.com/dpopsuev/origami/adapters/sqlite", false},
		{"origami.calibrate", "github.com/dpopsuev/origami/calibrate", false},
		{"unknown.module", "", true},
		{"origami", "", true},
		{"", "", true},
		{"ori;gami.bad", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.fqcn, func(t *testing.T) {
			got, err := reg.ResolveFQCN(tt.fqcn)
			if tt.err {
				if err == nil {
					t.Fatalf("expected error for %q, got %q", tt.fqcn, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("ResolveFQCN(%q) = %q, want %q", tt.fqcn, got, tt.want)
			}
		})
	}
}

func TestResolveProvider(t *testing.T) {
	reg := DefaultRegistry()

	tests := []struct {
		provider   string
		wantImport string
		wantSymbol string
		err        bool
	}{
		{"marbles.rca.CalibrateRunner", "github.com/dpopsuev/origami/marbles/rca", "CalibrateRunner", false},
		{"marbles.rca.AnalyzeFunc", "github.com/dpopsuev/origami/marbles/rca", "AnalyzeFunc", false},
		{"adapters.rp.Fetcher", "github.com/dpopsuev/origami/adapters/rp", "Fetcher", false},
		{"SingleWord", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			imp, sym, err := reg.ResolveProvider(tt.provider)
			if tt.err {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if imp != tt.wantImport {
				t.Errorf("import = %q, want %q", imp, tt.wantImport)
			}
			if sym != tt.wantSymbol {
				t.Errorf("symbol = %q, want %q", sym, tt.wantSymbol)
			}
		})
	}
}
