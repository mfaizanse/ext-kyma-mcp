package kyma

import (
	"fmt"
	"strings"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/kubernetes"
	"github.com/containers/kubernetes-mcp-server/pkg/mcplog"
	"github.com/containers/kubernetes-mcp-server/pkg/output"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/mfaizanse/ext-kyma-mcp/pkg/saphelp"
	"github.com/mfaizanse/ext-kyma-mcp/pkg/toolsets/common"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/ptr"
)

const (
	defaultKymaNamespace  = "kyma-system"
	defaultKymaName       = "default"
	defaultKymaAPIVersion = "operator.kyma-project.io/v1beta2"
	kymaKind              = "Kyma"
)

func initKyma() []api.ServerTool {
	return []api.ServerTool{
		{
			Tool: api.Tool{
				Name:        "kyma_get",
				Description: "Get the Kyma custom resource from the cluster",
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"namespace": {
							Type:        "string",
							Description: "Namespace of the Kyma CR (defaults to kyma-system)",
						},
						"name": {
							Type:        "string",
							Description: "Name of the Kyma CR (defaults to default)",
						},
						"apiVersion": {
							Type:        "string",
							Description: "Kyma API version (defaults to operator.kyma-project.io/v1beta2)",
						},
					},
				},
				Annotations: api.ToolAnnotations{
					Title:           "Kyma: Get",
					ReadOnlyHint:    ptr.To(true),
					DestructiveHint: ptr.To(false),
					OpenWorldHint:   ptr.To(true),
				},
			},
			Handler: kymaGet,
		},
		{
			Tool: api.Tool{
				Name:        "kyma_find_resource_version",
				Description: "Find the apiVersion for a Kyma resource kind (returns the first matched result)",
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"resourceKind": {
							Type:        "string",
							Description: "Kyma resource kind (e.g., Function, APIRule, TracePipeline) in PascalCase (singular)",
						},
					},
					Required: []string{"resourceKind"},
				},
				Annotations: api.ToolAnnotations{
					Title:           "Kyma: Resource Version",
					ReadOnlyHint:    ptr.To(true),
					DestructiveHint: ptr.To(false),
					OpenWorldHint:   ptr.To(true),
				},
			},
			Handler: kymaResourceVersion,
		},
		{
			Tool: api.Tool{
				Name:        "kyma_help_semantic_search",
				Description: "Search SAP Help Portal using semantic search and return relevant results",
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"query": {
							Type:        "string",
							Description: "Search query in detail for semantic search and include term kyma for better results",
						},
						"to": {
							Type:        "integer",
							Description: "Maximum number of results to return (defaults to 3)",
						},
						"locale": {
							Type:        "string",
							Description: "Locale hint for the search (defaults to en-US)",
						},
						"isExactMatch": {
							Type:        "boolean",
							Description: "Whether to require exact matches (defaults to false)",
						},
					},
					Required: []string{"query"},
				},
				Annotations: api.ToolAnnotations{
					Title:           "Kyma: SAP Help Semantic Search",
					ReadOnlyHint:    ptr.To(true),
					DestructiveHint: ptr.To(false),
					OpenWorldHint:   ptr.To(true),
				},
			},
			Handler: kymaHelpSemanticSearch,
		},
	}
}

func kymaGet(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	args := params.GetArguments()

	name, err := common.GetOptionalStringDefault(args, "name", defaultKymaName)
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}

	namespace, err := common.GetOptionalStringDefault(args, "namespace", defaultKymaNamespace)
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}

	apiVersion, err := common.GetOptionalStringDefault(args, "apiVersion", defaultKymaAPIVersion)
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}

	gv, err := schema.ParseGroupVersion(apiVersion)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("invalid apiVersion: %w", err)), nil
	}
	gvk := &schema.GroupVersionKind{Group: gv.Group, Version: gv.Version, Kind: kymaKind}

	ret, err := kubernetes.NewCore(params).ResourcesGet(params.Context, gvk, namespace, name)
	if err != nil {
		mcplog.HandleK8sError(params.Context, err, "kyma resource access")
		return api.NewToolCallResult("", fmt.Errorf("failed to get Kyma CR: %w", err)), nil
	}

	marshalled, err := output.MarshalYaml(ret)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to marshal Kyma CR: %w", err)), nil
	}

	return api.NewToolCallResult(strings.TrimSpace(marshalled), nil), nil
}

func kymaResourceVersion(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	args := params.GetArguments()
	kind, err := common.GetRequiredString(args, "resourceKind")
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}

	version, err := common.ResolveResourceVersion(params.DiscoveryClient(), kind)
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}

	return api.NewToolCallResult(version, nil), nil
}

func kymaHelpSemanticSearch(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	args := params.GetArguments()
	query, err := common.GetRequiredString(args, "query")
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}

	maxResults, err := common.GetOptionalInt(args, "to", 3)
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}
	if maxResults <= 0 {
		maxResults = 5
	}

	locale, err := common.GetOptionalStringDefault(args, "locale", "en-US")
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}

	isExactMatch, err := common.GetOptionalBool(args, "isExactMatch", false)
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}

	// Call the SAP Help semantic search function.
	content, err := saphelp.SAPHelpSemanticSearch(params.Context, query, maxResults, locale, isExactMatch)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("SAP Help semantic search failed: %w", err)), nil
	}
	return api.NewToolCallResult(content, nil), nil
}
