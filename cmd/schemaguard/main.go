package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/Evanskiplagat/SchemaGuard/internal/version"
)

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	for _, arg := range args {
		switch strings.TrimSpace(arg) {
		case "--help", "-h":
			printHelp(stdout)
			return nil
		case "--version":
			_, err := fmt.Fprintln(stdout, version.String())
			return err
		}
	}

	fs := flag.NewFlagSet("schemaguard", flag.ContinueOnError)
	fs.SetOutput(stderr)

	fs.Usage = func() {
		printHelp(stderr)
	}

	if err := fs.Parse(args); err != nil {
		return err
	}

	printHelp(stdout)
	return nil
}

func printHelp(w io.Writer) {
	fmt.Fprintf(w, "%s\n\n", version.String())
	fmt.Fprintln(w, "SchemaGuard checks OpenAPI specifications for breaking API changes.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  schemaguard [flags]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Flags:")
	fmt.Fprintln(w, "  --help       show help for SchemaGuard")
	fmt.Fprintln(w, "  --version    print the SchemaGuard version")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Planned commands:")
	fmt.Fprintln(w, "  schemaguard compare old.yaml new.yaml")
}
