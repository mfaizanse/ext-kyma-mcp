package common

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
)

// FilterEvents filters event maps using the provided predicate.
// If predicate is nil, the original list is returned.
func FilterEvents(events []map[string]any, predicate func(map[string]any) bool) []map[string]any {
	if predicate == nil {
		return events
	}
	filtered := make([]map[string]any, 0, len(events))
	for _, event := range events {
		if predicate(event) {
			filtered = append(filtered, event)
		}
	}
	return filtered
}

// ResolveResourceVersion finds the preferred apiVersion for the provided resource kind.
// Returns the first matched result from discovery.
func ResolveResourceVersion(client discovery.DiscoveryInterface, kind string) (string, error) {
	resourceLists, err := client.ServerPreferredResources()
	if err != nil && len(resourceLists) == 0 {
		return "", fmt.Errorf("failed to discover resources: %w", err)
	}

	for _, list := range resourceLists {
		gv, parseErr := schema.ParseGroupVersion(list.GroupVersion)
		if parseErr != nil {
			continue
		}
		for _, resource := range list.APIResources {
			if strings.EqualFold(resource.Kind, kind) {
				return gv.String(), nil
			}
		}
	}

	return "", fmt.Errorf("resource kind not found: %s", kind)
}
