package common

import (
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
