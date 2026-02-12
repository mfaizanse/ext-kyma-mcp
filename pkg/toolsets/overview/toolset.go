package overview

import (
	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/toolsets"
)

type Toolset struct{}

var _ api.Toolset = (*Toolset)(nil)

func (t *Toolset) GetName() string {
	return "overview"
}

func (t *Toolset) GetDescription() string {
	return "High-level overview tools for Kubernetes clusters"
}

func (t *Toolset) GetTools(api.Openshift) []api.ServerTool {
	return initOverview()
}

func (t *Toolset) GetPrompts() []api.ServerPrompt {
	return nil
}

func init() {
	toolsets.Register(&Toolset{})
}
