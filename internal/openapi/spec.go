package openapi

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

var openAPIVersionPattern = regexp.MustCompile(`^3\.\d+\.\d+(?:[-+].*)?$`)

// Spec is the subset of an OpenAPI document needed for validation and endpoint checks.
type Spec struct {
	OpenAPI string              `yaml:"openapi"`
	Info    *Info               `yaml:"info"`
	Paths   map[string]PathItem `yaml:"paths"`
}

type Info struct {
	Title   string `yaml:"title"`
	Version string `yaml:"version"`
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

type Operation struct {
	Responses map[string]Response `yaml:"responses"`
}

type Response struct {
	Description *string `yaml:"description"`
	Ref         string  `yaml:"$ref"`
	Content     map[string]MediaType `yaml:"content"`
}

type MediaType struct {
	Schema *Schema `yaml:"schema"`
}

type Schema struct {
	Ref        string             `yaml:"$ref"`
	Type       interface{}        `yaml:"type"`
	Properties map[string]*Schema `yaml:"properties"`
	Items      *Schema            `yaml:"items"`
}

func Load(path string) (*Spec, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %q: %w", path, err)
	}

	var document yaml.Node
	decoder := yaml.NewDecoder(bytes.NewReader(contents))
	if err := decoder.Decode(&document); err != nil {
		if errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("parse %q: document is empty", path)
		}
		return nil, fmt.Errorf("parse %q: %w", path, err)
	}
	var extra yaml.Node
	if err := decoder.Decode(&extra); err != io.EOF {
		if err != nil {
			return nil, fmt.Errorf("parse %q: %w", path, err)
		}
		return nil, fmt.Errorf("parse %q: multiple OpenAPI documents are not supported", path)
	}
	if err := resolveReferences(&document); err != nil {
		return nil, fmt.Errorf("validate %q: %w", path, err)
	}

	var spec Spec
	if err := document.Decode(&spec); err != nil {
		return nil, fmt.Errorf("parse %q: %w", path, err)
	}

	if !openAPIVersionPattern.MatchString(spec.OpenAPI) {
		return nil, fmt.Errorf("validate %q: expected an OpenAPI 3.x document", path)
	}
	if spec.Info == nil {
		return nil, fmt.Errorf("validate %q: missing required info object", path)
	}
	if strings.TrimSpace(spec.Info.Title) == "" {
		return nil, fmt.Errorf("validate %q: info.title is required", path)
	}
	if strings.TrimSpace(spec.Info.Version) == "" {
		return nil, fmt.Errorf("validate %q: info.version is required", path)
	}
	if spec.Paths == nil {
		return nil, fmt.Errorf("validate %q: missing paths object", path)
	}
	for path, pathItem := range spec.Paths {
		if !strings.HasPrefix(path, "/") {
			return nil, fmt.Errorf("validate %q: path %q must begin with /", path, path)
		}
		for _, method := range pathItem.Methods() {
			operation := pathItem.Operation(method)
			if len(operation.Responses) == 0 {
				return nil, fmt.Errorf("validate %q: %s %s is missing a responses object", path, method, path)
			}
			for status, response := range operation.Responses {
				if strings.TrimSpace(response.Ref) == "" && response.Description == nil {
					return nil, fmt.Errorf("validate %q: response %q for %s %s is missing description", path, status, method, path)
				}
			}
		}
	}

	return &spec, nil
}

func resolveReferences(document *yaml.Node) error {
	root := document
	if root.Kind == yaml.DocumentNode {
		if len(root.Content) != 1 {
			return errors.New("document must contain one root value")
		}
		root = root.Content[0]
	}
	return resolveNode(root, root, make(map[string]bool))
}

func resolveNode(node, root *yaml.Node, resolving map[string]bool) error {
	if node.Kind == yaml.MappingNode {
		if reference, ok := mappingValue(node, "$ref"); ok {
			if reference.Kind != yaml.ScalarNode || strings.TrimSpace(reference.Value) == "" {
				return errors.New("$ref must be a non-empty string")
			}
			ref := reference.Value
			if !strings.HasPrefix(ref, "#") {
				return fmt.Errorf("unsupported external $ref %q; only local references under #/components/schemas are supported", ref)
			}
			if !strings.HasPrefix(ref, "#/components/schemas/") || strings.TrimPrefix(ref, "#/components/schemas/") == "" {
				return fmt.Errorf("unsupported local $ref %q; only references under #/components/schemas are supported", ref)
			}
			if resolving[ref] {
				return fmt.Errorf("circular $ref %q", ref)
			}

			target, err := resolvePointer(root, ref)
			if err != nil {
				return err
			}
			resolving[ref] = true
			resolvedTarget := cloneNode(target)
			err = resolveNode(resolvedTarget, root, resolving)
			delete(resolving, ref)
			if err != nil {
				return err
			}
			*node = *resolvedTarget
			return nil
		}

		for index := 1; index < len(node.Content); index += 2 {
			if err := resolveNode(node.Content[index], root, resolving); err != nil {
				return err
			}
		}
		return nil
	}
	if node.Kind == yaml.SequenceNode {
		for _, child := range node.Content {
			if err := resolveNode(child, root, resolving); err != nil {
				return err
			}
		}
	}
	return nil
}

func mappingValue(node *yaml.Node, key string) (*yaml.Node, bool) {
	for index := 0; index+1 < len(node.Content); index += 2 {
		if node.Content[index].Value == key {
			return node.Content[index+1], true
		}
	}
	return nil, false
}

func resolvePointer(root *yaml.Node, ref string) (*yaml.Node, error) {
	current := root
	for _, token := range strings.Split(strings.TrimPrefix(ref, "#/"), "/") {
		decoded, err := decodePointerToken(token)
		if err != nil {
			return nil, fmt.Errorf("malformed $ref %q: %w", ref, err)
		}
		if current.Kind != yaml.MappingNode {
			return nil, fmt.Errorf("missing local $ref target %q", ref)
		}
		var found bool
		for index := 0; index+1 < len(current.Content); index += 2 {
			if current.Content[index].Value == decoded {
				current = current.Content[index+1]
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("missing local $ref target %q", ref)
		}
	}
	return current, nil
}

func decodePointerToken(token string) (string, error) {
	var builder strings.Builder
	for index := 0; index < len(token); index++ {
		if token[index] != '~' {
			builder.WriteByte(token[index])
			continue
		}
		if index+1 >= len(token) || (token[index+1] != '0' && token[index+1] != '1') {
			return "", errors.New("invalid JSON Pointer escape")
		}
		if token[index+1] == '0' {
			builder.WriteByte('~')
		} else {
			builder.WriteByte('/')
		}
		index++
	}
	return builder.String(), nil
}

func cloneNode(node *yaml.Node) *yaml.Node {
	clone := *node
	clone.Content = make([]*yaml.Node, len(node.Content))
	for index, child := range node.Content {
		clone.Content[index] = cloneNode(child)
	}
	return &clone
}

func (p PathItem) Operation(method string) *Operation {
	switch method {
	case "DELETE":
		return p.Delete
	case "GET":
		return p.Get
	case "HEAD":
		return p.Head
	case "OPTIONS":
		return p.Options
	case "PATCH":
		return p.Patch
	case "POST":
		return p.Post
	case "PUT":
		return p.Put
	case "TRACE":
		return p.Trace
	default:
		return nil
	}
}

// Types returns the schema's declared types in a normalized, sorted form.
// OpenAPI 3.0 uses a string, while OpenAPI 3.1 also permits an array of types.
func (s *Schema) Types() []string {
	if s == nil {
		return nil
	}

	var types []string
	switch value := s.Type.(type) {
	case string:
		types = append(types, value)
	case []interface{}:
		for _, item := range value {
			if typeName, ok := item.(string); ok {
				types = append(types, typeName)
			}
		}
	}
	sort.Strings(types)
	return types
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
