package openapi

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Spec is the subset of an OpenAPI document needed for endpoint compatibility checks.
type Spec struct {
	OpenAPI string              `yaml:"openapi"`
	Paths   map[string]PathItem `yaml:"paths"`
}

type PathItem struct {
	Delete  *Operation `yaml:"delete"`
	Get     *Operation `yaml:"get"`
	Head    *Operation `yaml:"head"`
	Options *Operation `yaml:"options"`
	Patch   *Operation `yaml:"patch"`
	Post    *Operation `yaml:"post"`
	Put     *Operation `yaml:"put"`
	Trace   *Operation `yaml:"trace"`
}

// Operation is intentionally empty: its presence represents a supported HTTP operation.
type Operation struct{}

func Load(path string) (*Spec, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %q: %w", path, err)
	}

	var spec Spec
	if err := yaml.Unmarshal(contents, &spec); err != nil {
		return nil, fmt.Errorf("parse %q: %w", path, err)
	}
	if !strings.HasPrefix(spec.OpenAPI, "3.") {
		return nil, fmt.Errorf("validate %q: expected an OpenAPI 3.x document", path)
	}
	if spec.Paths == nil {
		return nil, fmt.Errorf("validate %q: missing paths object", path)
	}
	for path := range spec.Paths {
		if !strings.HasPrefix(path, "/") {
			return nil, fmt.Errorf("validate %q: path %q must begin with /", path, path)
		}
	}

	return &spec, nil
}

func (p PathItem) Methods() []string {
	methods := make([]string, 0, 8)
	for method, operation := range map[string]*Operation{
		"DELETE":  p.Delete,
		"GET":     p.Get,
		"HEAD":    p.Head,
		"OPTIONS": p.Options,
		"PATCH":   p.Patch,
		"POST":    p.Post,
		"PUT":     p.Put,
		"TRACE":   p.Trace,
	} {
		if operation != nil {
			methods = append(methods, method)
		}
	}
	sort.Strings(methods)
	return methods
}
