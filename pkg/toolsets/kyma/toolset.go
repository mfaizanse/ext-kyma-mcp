package kyma

import (
	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/toolsets"
)

type Toolset struct{}

var _ api.Toolset = (*Toolset)(nil)

func (t *Toolset) GetName() string {
	return "kyma"
}

func (t *Toolset) GetDescription() string {
	return "Kyma-specific tools for interacting with Kyma custom resources"
}

func (t *Toolset) GetTools(api.Openshift) []api.ServerTool {
	return initKyma()
}

func (t *Toolset) GetPrompts() []api.ServerPrompt {
	return nil
}

func init() {
	toolsets.Register(&Toolset{})
}
