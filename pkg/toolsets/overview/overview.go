package overview

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/kubernetes"
	"github.com/containers/kubernetes-mcp-server/pkg/mcplog"
	"github.com/containers/kubernetes-mcp-server/pkg/output"
	"github.com/google/jsonschema-go/jsonschema"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/kubectl/pkg/metricsutil"
	"k8s.io/utils/ptr"
)

const (
	clusterKind   = "cluster"
	namespaceKind = "namespace"
)

func initOverview() []api.ServerTool {
	return []api.ServerTool{
		{
			Tool: api.Tool{
				Name:        "overview_cluster_version",
				Description: "Get the Kubernetes cluster version",
				InputSchema: &jsonschema.Schema{
					Type:       "object",
					Properties: map[string]*jsonschema.Schema{},
				},
				Annotations: api.ToolAnnotations{
					Title:           "Overview: Cluster Version",
					ReadOnlyHint:    ptr.To(true),
					DestructiveHint: ptr.To(false),
					OpenWorldHint:   ptr.To(true),
				},
			},
			Handler: overviewClusterVersion,
		},
		{
			Tool: api.Tool{
				Name:        "overview_relevant_context",
				Description: "Fetch relevant Kubernetes context for a cluster, namespace, or specific resource (use kind=cluster as a logical signal for whole-cluster context)",
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"kind": {
							Type:        "string",
							Description: "Kind of the context to fetch (cluster, namespace, or a Kubernetes resource kind)",
						},
						"namespace": {
							Type:        "string",
							Description: "Namespace to scope the request (optional)",
						},
						"name": {
							Type:        "string",
							Description: "Name of the resource (required for resource context)",
						},
						"apiVersion": {
							Type:        "string",
							Description: "apiVersion of the resource (required for resource context)",
						},
					},
					Required: []string{"kind"},
				},
				Annotations: api.ToolAnnotations{
					Title:           "Overview: Relevant Context",
					ReadOnlyHint:    ptr.To(true),
					DestructiveHint: ptr.To(false),
					OpenWorldHint:   ptr.To(true),
				},
			},
			Handler: overviewRelevantContext,
		},
	}
}

func overviewClusterVersion(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	versionInfo, err := params.DiscoveryClient().ServerVersion()
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get cluster version: %w", err)), nil
	}

	payload, err := output.MarshalYaml(versionInfo)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to marshal cluster version: %w", err)), nil
	}

	return api.NewToolCallResult(strings.TrimSpace(payload), nil), nil
}

func overviewRelevantContext(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	args := params.GetArguments()
	kind, err := getRequiredString(args, "kind")
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}

	namespace, err := getOptionalString(args, "namespace")
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}

	name, err := getOptionalString(args, "name")
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}

	apiVersion, err := getOptionalString(args, "apiVersion")
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}

	switch {
	case namespace == "" && strings.EqualFold(kind, clusterKind):
		return clusterOverviewContext(params)
	case namespace != "" && strings.EqualFold(kind, namespaceKind):
		return namespaceOverviewContext(params, namespace)
	case apiVersion != "":
		if name == "" {
			return api.NewToolCallResult("", fmt.Errorf("name is required for resource context")), nil
		}
		return resourceOverviewContext(params, apiVersion, kind, namespace, name)
	default:
		return api.NewToolCallResult("", fmt.Errorf("invalid arguments: provide kind=cluster, kind=namespace with namespace, or kind/apiVersion/name for a resource")), nil
	}
}

func clusterOverviewContext(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	core := kubernetes.NewCore(params)
	options := api.ListOptions{ListOptions: metav1.ListOptions{FieldSelector: "status.phase!=Running"}}

	pods, err := core.PodsListInAllNamespaces(params, options)
	if err != nil {
		mcplog.HandleK8sError(params.Context, err, "pods listing")
		return api.NewToolCallResult("", fmt.Errorf("failed to list non-running pods: %w", err)), nil
	}
	podsYaml, err := output.MarshalYaml(pods)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to marshal pod list: %w", err)), nil
	}

	metrics, err := formatNodeMetrics(params, core)
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}

	warningEvents, err := listWarningEvents(params, "")
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}

	kymaStatus := "# Kyma CR Status (YAML)\n# Kyma CR not found or unavailable"
	if status, statusErr := fetchKymaStatus(params); statusErr == nil && strings.TrimSpace(status) != "" {
		kymaStatus = strings.Join([]string{"# Kyma CR Status (YAML)", status}, "\n")
	}

	content := strings.Join([]string{
		"# Not running Pods (YAML)",
		strings.TrimSpace(podsYaml),
		"# Node Metrics",
		strings.TrimSpace(metrics),
		"# Warning Events (YAML)",
		warningEvents,
		kymaStatus,
	}, "\n")
	return api.NewToolCallResult(content, nil), nil
}

func namespaceOverviewContext(params api.ToolHandlerParams, namespace string) (*api.ToolCallResult, error) {
	warningEvents, err := listWarningEvents(params, namespace)
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}
	content := strings.Join([]string{
		"# Warning Events (YAML)",
		warningEvents,
	}, "\n")
	return api.NewToolCallResult(content, nil), nil
}

func resourceOverviewContext(params api.ToolHandlerParams, apiVersion, kind, namespace, name string) (*api.ToolCallResult, error) {
	gv, err := schema.ParseGroupVersion(apiVersion)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("invalid apiVersion: %w", err)), nil
	}
	gvk := &schema.GroupVersionKind{Group: gv.Group, Version: gv.Version, Kind: kind}

	resource, err := kubernetes.NewCore(params).ResourcesGet(params.Context, gvk, namespace, name)
	if err != nil {
		mcplog.HandleK8sError(params.Context, err, "resource access")
		return api.NewToolCallResult("", fmt.Errorf("failed to get resource: %w", err)), nil
	}
	resourceYaml, err := output.MarshalYaml(resource)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to marshal resource: %w", err)), nil
	}

	resourceEvents, err := listEventsForResource(params, namespace, kind, name)
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}

	content := strings.Join([]string{
		"# Resource (YAML)",
		strings.TrimSpace(resourceYaml),
		"# Resource Events (YAML)",
		resourceEvents,
	}, "\n")
	return api.NewToolCallResult(content, nil), nil
}

func listWarningEvents(params api.ToolHandlerParams, namespace string) (string, error) {
	events, err := kubernetes.NewCore(params).EventsList(params.Context, namespace)
	if err != nil {
		mcplog.HandleK8sError(params.Context, err, "events listing")
		return "", fmt.Errorf("failed to list events: %w", err)
	}
	filtered := filterEvents(events, func(event map[string]any) bool {
		value, ok := event["Type"].(string)
		return ok && strings.EqualFold(value, "Warning")
	})
	if len(filtered) == 0 {
		return "# No warning events found", nil
	}
	yamlEvents, err := output.MarshalYaml(filtered)
	if err != nil {
		return "", fmt.Errorf("failed to marshal warning events: %w", err)
	}
	return strings.TrimSpace(yamlEvents), nil
}

func listEventsForResource(params api.ToolHandlerParams, namespace, kind, name string) (string, error) {
	events, err := kubernetes.NewCore(params).EventsList(params.Context, namespace)
	if err != nil {
		mcplog.HandleK8sError(params.Context, err, "events listing")
		return "", fmt.Errorf("failed to list events: %w", err)
	}
	filtered := filterEvents(events, func(event map[string]any) bool {
		involved, ok := event["InvolvedObject"].(map[string]string)
		if !ok {
			return false
		}
		return strings.EqualFold(involved["Kind"], kind) && strings.EqualFold(involved["Name"], name)
	})
	if len(filtered) == 0 {
		return "# No events found for resource", nil
	}
	yamlEvents, err := output.MarshalYaml(filtered)
	if err != nil {
		return "", fmt.Errorf("failed to marshal resource events: %w", err)
	}
	return strings.TrimSpace(yamlEvents), nil
}

func fetchKymaStatus(params api.ToolHandlerParams) (string, error) {
	gvk := &schema.GroupVersionKind{Group: "operator.kyma-project.io", Version: "v1beta2", Kind: "Kyma"}
	resource, err := kubernetes.NewCore(params).ResourcesGet(params.Context, gvk, "kyma-system", "default")
	if err != nil {
		return "", err
	}
	status, found, err := unstructured.NestedFieldCopy(resource.Object, "status")
	if err != nil || !found {
		return "", err
	}
	marshalled, err := output.MarshalYaml(status)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(marshalled), nil
}

func formatNodeMetrics(params api.ToolHandlerParams, core *kubernetes.Core) (string, error) {
	nodeMetrics, err := core.NodesTop(params, api.NodesTopOptions{})
	if err != nil {
		mcplog.HandleK8sError(params.Context, err, "node metrics access")
		return "", fmt.Errorf("failed to get node metrics: %w", err)
	}

	nodeList, err := params.CoreV1().Nodes().List(params, metav1.ListOptions{})
	if err != nil {
		mcplog.HandleK8sError(params.Context, err, "node listing")
		return "", fmt.Errorf("failed to list nodes: %w", err)
	}

	availableResources := make(map[string]v1.ResourceList)
	for _, node := range nodeList.Items {
		availableResources[node.Name] = node.Status.Allocatable
		if node.Status.NodeInfo.Swap != nil && node.Status.NodeInfo.Swap.Capacity != nil {
			swapCapacity := *node.Status.NodeInfo.Swap.Capacity
			availableResources[node.Name]["swap"] = *resource.NewQuantity(swapCapacity, resource.BinarySI)
		}
	}

	buf := new(bytes.Buffer)
	printer := metricsutil.NewTopCmdPrinter(buf, true)
	if err := printer.PrintNodeMetrics(nodeMetrics.Items, availableResources, false, ""); err != nil {
		return "", fmt.Errorf("failed to print node metrics: %w", err)
	}
	return strings.TrimSpace(buf.String()), nil
}

func filterEvents(events []map[string]any, predicate func(map[string]any) bool) []map[string]any {
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

func getRequiredString(args map[string]any, key string) (string, error) {
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

func getOptionalString(args map[string]any, key string) (string, error) {
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
