package compare

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Evanskiplagat/SchemaGuard/internal/openapi"
)

// BreakingChange describes a backwards-incompatible API change.
type BreakingChange struct {
	Message string
}

func Specs(oldSpec, newSpec *openapi.Spec) []BreakingChange {
	changes := make([]BreakingChange, 0)
	for path, oldPath := range oldSpec.Paths {
		newPath, exists := newSpec.Paths[path]
		if !exists {
			changes = append(changes, BreakingChange{Message: fmt.Sprintf("removed path %s", path)})
			continue
		}

		newMethods := make(map[string]struct{})
		for _, method := range newPath.Methods() {
			newMethods[method] = struct{}{}
		}
		for _, method := range oldPath.Methods() {
			if _, exists := newMethods[method]; !exists {
				changes = append(changes, BreakingChange{Message: fmt.Sprintf("removed operation %s %s", method, path)})
				continue
			}
			changes = append(changes, compareResponses(path, method, oldPath.Operation(method), newPath.Operation(method))...)
		}
	}

	sort.Slice(changes, func(i, j int) bool {
		return changes[i].Message < changes[j].Message
	})
	return changes
}

func compareResponses(path, method string, oldOperation, newOperation *openapi.Operation) []BreakingChange {
	changes := make([]BreakingChange, 0)
	for status, oldResponse := range oldOperation.Responses {
		newResponse, exists := newOperation.Responses[status]
		if !exists {
			changes = append(changes, BreakingChange{Message: fmt.Sprintf("removed response %s for %s %s", status, method, path)})
			continue
		}
		changes = append(changes, compareResponseContent(path, method, status, oldResponse, newResponse)...)
	}
	return changes
}

func compareResponseContent(path, method, status string, oldResponse, newResponse openapi.Response) []BreakingChange {
	changes := make([]BreakingChange, 0)
	for mediaType, oldContent := range oldResponse.Content {
		newContent, exists := newResponse.Content[mediaType]
		if !exists {
			changes = append(changes, BreakingChange{Message: fmt.Sprintf("removed response media type %s for %s %s %s", mediaType, status, method, path)})
			continue
		}
		if oldContent.Schema != nil && newContent.Schema == nil {
			changes = append(changes, BreakingChange{Message: fmt.Sprintf("removed response schema for %s %s %s (%s)", status, method, path, mediaType)})
			continue
		}
		compareSchema(&changes, path, method, status, mediaType, "", oldContent.Schema, newContent.Schema)
	}
	return changes
}

func compareSchema(changes *[]BreakingChange, path, method, status, mediaType, propertyPath string, oldSchema, newSchema *openapi.Schema) {
	if oldSchema == nil || newSchema == nil {
		return
	}

	oldTypes := oldSchema.Types()
	newTypes := newSchema.Types()
	if len(oldTypes) > 0 && len(newTypes) > 0 && strings.Join(oldTypes, ",") != strings.Join(newTypes, ",") && propertyPath != "" {
		*changes = append(*changes, BreakingChange{Message: fmt.Sprintf("changed response property type from %s to %s for %s in %s %s %s (%s)", strings.Join(oldTypes, " or "), strings.Join(newTypes, " or "), propertyPath, status, method, path, mediaType)})
	}

	for property, oldPropertySchema := range oldSchema.Properties {
		currentPath := property
		if propertyPath != "" {
			currentPath = propertyPath + "." + property
		}
		newPropertySchema, exists := newSchema.Properties[property]
		if !exists || newPropertySchema == nil {
			*changes = append(*changes, BreakingChange{Message: fmt.Sprintf("removed response property %s from %s %s %s (%s)", currentPath, status, method, path, mediaType)})
			continue
		}
		compareSchema(changes, path, method, status, mediaType, currentPath, oldPropertySchema, newPropertySchema)
	}

	if oldSchema.Items != nil && newSchema.Items != nil {
		itemPath := "items"
		if propertyPath != "" {
			itemPath = propertyPath + ".items"
		}
		compareSchema(changes, path, method, status, mediaType, itemPath, oldSchema.Items, newSchema.Items)
	}
}
