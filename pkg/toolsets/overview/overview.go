package overview

import (
	"fmt"
	"strings"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/output"
	"github.com/google/jsonschema-go/jsonschema"
	"k8s.io/utils/ptr"
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
