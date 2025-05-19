package k8s

import (
	"fmt"
	"strings"
)

func UpdateField(obj map[string]interface{}, parts []string, value interface{}) error {
	if len(parts) == 1 {
		obj[parts[0]] = value
		return nil
	}

	nextPart := parts[0]
	nextObj, ok := obj[nextPart].(map[string]interface{})
	if !ok {
		return fmt.Errorf("field path does not exist: %s", strings.Join(parts, "."))
	}

	return UpdateField(nextObj, parts[1:], value)
}
