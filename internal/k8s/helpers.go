package k8s

import (
	"fmt"
	"strings"
)

// UpdateField goes through obj and updates desired part by setting it to value.
func UpdateField(obj map[string]any, parts []string, value any) error {
	if len(parts) == 1 {
		obj[parts[0]] = value
		return nil
	}

	nextPart := parts[0]
	nextObj, ok := obj[nextPart].(map[string]any)
	if !ok {
		return fmt.Errorf("field path does not exist: %s", strings.Join(parts, "."))
	}

	return UpdateField(nextObj, parts[1:], value)
}
