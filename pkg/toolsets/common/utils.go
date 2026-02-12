package common

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
