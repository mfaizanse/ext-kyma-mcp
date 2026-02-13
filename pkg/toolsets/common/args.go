package common

import (
	"encoding/json"
	"fmt"
	"strings"
)

func GetRequiredString(args map[string]any, key string) (string, error) {
	value, ok := args[key]
	if !ok || value == nil {
		return "", fmt.Errorf("%s is required", key)
	}
	strValue, ok := value.(string)
	if !ok || strings.TrimSpace(strValue) == "" {
		return "", fmt.Errorf("%s is required", key)
	}
	return strings.TrimSpace(strValue), nil
}

func GetOptionalString(args map[string]any, key string) (string, error) {
	value, ok := args[key]
	if !ok || value == nil {
		return "", nil
	}
	strValue, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("%s is not a string", key)
	}
	return strings.TrimSpace(strValue), nil
}

func GetOptionalStringDefault(args map[string]any, key, defaultValue string) (string, error) {
	value, ok := args[key]
	if !ok || value == nil {
		return defaultValue, nil
	}
	strValue, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("%s is not a string", key)
	}
	trimmed := strings.TrimSpace(strValue)
	if trimmed == "" {
		return defaultValue, nil
	}
	return trimmed, nil
}

func GetOptionalInt(args map[string]any, key string, defaultValue int) (int, error) {
	value, ok := args[key]
	if !ok || value == nil {
		return defaultValue, nil
	}
	switch typed := value.(type) {
	case float64:
		return int(typed), nil
	case float32:
		return int(typed), nil
	case int:
		return typed, nil
	case int64:
		return int(typed), nil
	case json.Number:
		parsed, err := typed.Int64()
		if err != nil {
			return 0, fmt.Errorf("%s is not a valid integer", key)
		}
		return int(parsed), nil
	default:
		return 0, fmt.Errorf("%s is not a valid integer", key)
	}
}

func GetOptionalBool(args map[string]any, key string, defaultValue bool) (bool, error) {
	value, ok := args[key]
	if !ok || value == nil {
		return defaultValue, nil
	}
	boolValue, ok := value.(bool)
	if !ok {
		return false, fmt.Errorf("%s is not a boolean", key)
	}
	return boolValue, nil
}
