package cmd

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

func stringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case json.Number:
		return typed.String()
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(typed), 'f', -1, 32)
	case int:
		return strconv.Itoa(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	case bool:
		return strconv.FormatBool(typed)
	case nil:
		return ""
	default:
		return fmt.Sprint(typed)
	}
}

func nested(object map[string]any, path ...string) any {
	var current any = object
	for _, key := range path {
		mapped, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		current = mapped[key]
	}
	return current
}

func contactName(contact map[string]any) string {
	if name := stringValue(nested(contact, "company", "name")); name != "" {
		return name
	}
	parts := []string{
		stringValue(nested(contact, "person", "firstName")),
		stringValue(nested(contact, "person", "lastName")),
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}

func contactNumber(contact map[string]any) string {
	if number := stringValue(nested(contact, "roles", "customer", "number")); number != "" {
		return number
	}
	return stringValue(nested(contact, "roles", "vendor", "number"))
}

func mergeObjects(base, changes map[string]any) map[string]any {
	for key, change := range changes {
		changeObject, changeIsObject := change.(map[string]any)
		baseObject, baseIsObject := base[key].(map[string]any)
		if changeIsObject && baseIsObject {
			base[key] = mergeObjects(baseObject, changeObject)
			continue
		}
		base[key] = change
	}
	return base
}
