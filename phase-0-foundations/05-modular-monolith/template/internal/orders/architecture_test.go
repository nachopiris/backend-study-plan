package orders_test

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

// TestArchitecture_OrdersDoesNotImportAdapterInternals is the architecture
// test the README refers to: it parses every .go file in this package
// (internal/orders) and fails if any of them imports the concrete in-memory
// adapters of inventory or payments, instead of depending only on their
// ports (the inventory and payments root packages).
//
// This is what keeps orders swappable: an adapter can be replaced (in-memory
// today, a real database or an HTTP client tomorrow) without ever touching
// orders, precisely because orders never knew the adapter's package existed.
func TestArchitecture_OrdersDoesNotImportAdapterInternals(t *testing.T) {
	forbiddenSuffixes := []string{
		"/internal/inventory/memory",
		"/internal/payments/memory",
	}

	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("listing files in internal/orders: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("expected to find .go files in internal/orders, found none")
	}

	fset := token.NewFileSet()
	for _, file := range files {
		f, err := parser.ParseFile(fset, file, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parsing %s: %v", file, err)
		}
		for _, imp := range f.Imports {
			path := strings.Trim(imp.Path.Value, `"`)
			for _, forbidden := range forbiddenSuffixes {
				if strings.HasSuffix(path, forbidden) {
					t.Errorf(
						"%s imports %q; orders must depend only on the inventory/payments ports, never on their concrete adapters",
						file, path,
					)
				}
			}
		}
	}
}
