package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunVersion(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := run([]string{"--version"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	got := strings.TrimSpace(stdout.String())
	want := "SchemaGuard 0.1.0-dev"
	if got != want {
		t.Fatalf("unexpected version output: got %q want %q", got, want)
	}

	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestRunHelp(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := run([]string{"--help"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	output := stdout.String()
	for _, fragment := range []string{
		"SchemaGuard checks OpenAPI specifications for breaking API changes.",
		"schemaguard compare old.yaml new.yaml",
	} {
		if !strings.Contains(output, fragment) {
			t.Fatalf("help output missing %q in %q", fragment, output)
		}
	}

	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}
