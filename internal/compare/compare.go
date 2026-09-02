package compare

import (
	"fmt"
	"sort"

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
			}
		}
	}

	sort.Slice(changes, func(i, j int) bool {
		return changes[i].Message < changes[j].Message
	})
	return changes
}
