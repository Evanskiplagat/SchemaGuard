package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
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
		"compare <old-spec> <new-spec>",
		"1  breaking changes found",
	} {
		if !strings.Contains(output, fragment) {
			t.Fatalf("help output missing %q in %q", fragment, output)
		}
	}

	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestRunCompareCompatible(t *testing.T) {
	directory := t.TempDir()
	oldPath := writeSpec(t, directory, "old.yaml", validYAML("  /pets:\n    get:\n      responses:\n        '200':\n          description: OK\n"))
	newPath := writeSpec(t, directory, "new.yaml", validYAML("  /pets:\n    get:\n      responses:\n        '200':\n          description: OK\n    post:\n      responses:\n        '201':\n          description: Created\n"))

	var stdout bytes.Buffer
	err := run([]string{"compare", oldPath, newPath}, &stdout, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}
	if got := strings.TrimSpace(stdout.String()); got != "Compatible: no breaking changes found." {
		t.Fatalf("unexpected output: %q", got)
	}
}

func TestRunCompareBreakingChanges(t *testing.T) {
	directory := t.TempDir()
	oldPath := writeSpec(t, directory, "old.yaml", validYAML("  /pets:\n    get:\n      responses:\n        '200':\n          description: OK\n    post:\n      responses:\n        '201':\n          description: Created\n  /users:\n    get:\n      responses:\n        '200':\n          description: OK\n"))
	newPath := writeSpec(t, directory, "new.yaml", validYAML("  /pets:\n    get:\n      responses:\n        '200':\n          description: OK\n"))

	var stdout bytes.Buffer
	err := run([]string{"compare", oldPath, newPath}, &stdout, &bytes.Buffer{})
	if exitCode(err) != exitBreakingChange {
		t.Fatalf("expected exit code %d, got error %v", exitBreakingChange, err)
	}
	for _, fragment := range []string{"removed operation POST /pets", "removed path /users"} {
		if !strings.Contains(stdout.String(), fragment) {
			t.Fatalf("output missing %q: %s", fragment, stdout.String())
		}
	}
}

func TestRunCompareResponseBreakingChanges(t *testing.T) {
	directory := t.TempDir()
	oldPath := writeSpec(t, directory, "old.yaml", validYAML("  /pets:\n    get:\n      responses:\n        '200':\n          description: OK\n          content:\n            application/json:\n              schema:\n                type: object\n                properties:\n                  id:\n                    type: string\n                  owner:\n                    type: object\n                    properties:\n                      name:\n                        type: string\n            text/plain:\n              schema:\n                type: string\n        '404':\n          description: Not found\n"))
	newPath := writeSpec(t, directory, "new.yaml", validYAML("  /pets:\n    get:\n      responses:\n        '200':\n          description: OK\n          content:\n            application/json:\n              schema:\n                type: object\n                properties:\n                  id:\n                    type: integer\n                  owner:\n                    type: object\n                    properties: {}\n"))

	var stdout bytes.Buffer
	err := run([]string{"compare", oldPath, newPath}, &stdout, &bytes.Buffer{})
	if exitCode(err) != exitBreakingChange {
		t.Fatalf("expected exit code %d, got error %v", exitBreakingChange, err)
	}
	for _, fragment := range []string{
		"removed response 404 for GET /pets",
		"removed response media type text/plain for 200 GET /pets",
		"changed response property type from string to integer for id in 200 GET /pets (application/json)",
		"removed response property owner.name from 200 GET /pets (application/json)",
	} {
		if !strings.Contains(stdout.String(), fragment) {
			t.Fatalf("output missing %q: %s", fragment, stdout.String())
		}
	}
}

func TestRunCompareInvalidSpec(t *testing.T) {
	directory := t.TempDir()
	oldPath := writeSpec(t, directory, "old.yaml", "openapi: 2.0\ninfo:\n  title: Old\n  version: 1.0.0\npaths: {}\n")
	newPath := writeSpec(t, directory, "new.yaml", validYAML(""))

	err := run([]string{"compare", oldPath, newPath}, &bytes.Buffer{}, &bytes.Buffer{})
	if exitCode(err) != exitInvalidInput {
		t.Fatalf("expected exit code %d, got error %v", exitInvalidInput, err)
	}
}

func TestRunCompareJSON(t *testing.T) {
	directory := t.TempDir()
	oldPath := writeSpec(t, directory, "old.json", `{"openapi":"3.1.0","info":{"title":"Pets","version":"1.0.0"},"paths":{"/pets":{"get":{"responses":{"200":{"description":"OK"}}}}}}`)
	newPath := writeSpec(t, directory, "new.json", `{"openapi":"3.1.0","info":{"title":"Pets","version":"1.1.0"},"paths":{"/pets":{"get":{"responses":{"200":{"description":"OK"}}}}}}`)

	err := run([]string{"compare", oldPath, newPath}, &bytes.Buffer{}, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("JSON specs should be accepted: %v", err)
	}
}

func TestRunCompareInvalidOperation(t *testing.T) {
	directory := t.TempDir()
	oldPath := writeSpec(t, directory, "old.yaml", validYAML("  /pets:\n    get: {}\n"))
	newPath := writeSpec(t, directory, "new.yaml", validYAML(""))

	err := run([]string{"compare", oldPath, newPath}, &bytes.Buffer{}, &bytes.Buffer{})
	if exitCode(err) != exitInvalidInput || !strings.Contains(err.Error(), "missing a responses object") {
		t.Fatalf("expected a clear validation error, got %v", err)
	}
}

func TestRunCompareInvalidYAML(t *testing.T) {
	directory := t.TempDir()
	oldPath := writeSpec(t, directory, "old.yaml", "openapi: [\n")
	newPath := writeSpec(t, directory, "new.yaml", validYAML(""))

	err := run([]string{"compare", oldPath, newPath}, &bytes.Buffer{}, &bytes.Buffer{})
	if exitCode(err) != exitInvalidInput || !strings.Contains(err.Error(), "parse") {
		t.Fatalf("expected a parse error, got %v", err)
	}
}

func validYAML(paths string) string {
	return "openapi: 3.0.3\ninfo:\n  title: Example API\n  version: 1.0.0\npaths:\n" + paths
}

func writeSpec(t *testing.T, directory, name, contents string) string {
	t.Helper()
	path := filepath.Join(directory, name)
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write spec: %v", err)
	}
	return path
}

func exitCode(err error) int {
	var commandErr *commandError
	if !errors.As(err, &commandErr) {
		return 0
	}
	return commandErr.code
}
