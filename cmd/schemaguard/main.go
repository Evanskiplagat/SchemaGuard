package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/Evanskiplagat/SchemaGuard/internal/compare"
	"github.com/Evanskiplagat/SchemaGuard/internal/openapi"
	"github.com/Evanskiplagat/SchemaGuard/internal/version"
)

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		var commandErr *commandError
		if errors.As(err, &commandErr) {
			os.Exit(commandErr.code)
		}
		os.Exit(exitInvalidInput)
	}
}

const (
	exitBreakingChange = 1
	exitInvalidInput   = 2
)

type commandError struct {
	code int
	err  error
}

func (e *commandError) Error() string { return e.err.Error() }

func run(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		printHelp(stdout)
		return nil
	}

	switch strings.TrimSpace(args[0]) {
	case "--help", "-h", "help":
		printHelp(stdout)
		return nil
	case "--version":
		_, err := fmt.Fprintln(stdout, version.String())
		return err
	case "compare":
		return runCompare(args[1:], stdout)
	default:
		return &commandError{code: exitInvalidInput, err: fmt.Errorf("unknown command %q; run schemaguard --help", args[0])}
	}
}

func runCompare(args []string, stdout io.Writer) error {
	if len(args) != 2 {
		return &commandError{code: exitInvalidInput, err: errors.New("usage: schemaguard compare <old-spec> <new-spec>")}
	}

	oldSpec, err := openapi.Load(args[0])
	if err != nil {
		return &commandError{code: exitInvalidInput, err: err}
	}
	newSpec, err := openapi.Load(args[1])
	if err != nil {
		return &commandError{code: exitInvalidInput, err: err}
	}

	changes := compare.Specs(oldSpec, newSpec)
	if len(changes) == 0 {
		_, err := fmt.Fprintln(stdout, "Compatible: no breaking changes found.")
		return err
	}

	fmt.Fprintln(stdout, "Breaking changes found:")
	for _, change := range changes {
		fmt.Fprintf(stdout, "- %s\n", change.Message)
	}
	return &commandError{code: exitBreakingChange, err: errors.New("compatibility check failed")}
}

func printHelp(w io.Writer) {
	fmt.Fprintf(w, "%s\n\n", version.String())
	fmt.Fprintln(w, "SchemaGuard checks OpenAPI specifications for breaking API changes.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  schemaguard <command>")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Commands:")
	fmt.Fprintln(w, "  compare <old-spec> <new-spec>  check OpenAPI 3.x specs for removed paths and operations")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Options:")
	fmt.Fprintln(w, "  --help       show help for SchemaGuard")
	fmt.Fprintln(w, "  --version    print the SchemaGuard version")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Exit codes:")
	fmt.Fprintln(w, "  0  compatible")
	fmt.Fprintln(w, "  1  breaking changes found")
	fmt.Fprintln(w, "  2  invalid command or specification")
}
