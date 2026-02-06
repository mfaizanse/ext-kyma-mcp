package kubernetes

import (
	"context"
	"errors"
	"fmt"

	kmsapi "github.com/containers/kubernetes-mcp-server/pkg/api"
	kmskubernetes "github.com/containers/kubernetes-mcp-server/pkg/kubernetes"
	"github.com/mfaizanse/ext-kyma-mcp/pkg/config"
	authenticationv1api "k8s.io/api/authentication/v1"
	"k8s.io/klog/v2"
)

// AuthHeadersClusterProvider implements Provider for authentication via request headers.
// This provider requires users to provide authentication tokens via request headers.
// It uses cluster connection details from configuration but does not use any
// authentication credentials from kubeconfig files.
type AuthHeadersClusterProvider struct {
	config kmsapi.BaseConfig
}

var _ kmskubernetes.Provider = &AuthHeadersClusterProvider{}

func init() {
	kmskubernetes.RegisterProvider(config.ClusterProviderAuthHeaders, newAuthHeadersClusterProvider)
}

// newAuthHeadersClusterProvider creates a provider that requires header-based authentication.
// Users must provide tokens via request headers (server URL, Token or client certificate and key).
func newAuthHeadersClusterProvider(cfg kmsapi.BaseConfig) (kmskubernetes.Provider, error) {
	ret := &AuthHeadersClusterProvider{config: cfg}
	if err := ret.reset(); err != nil {
		return nil, err
	}
	return ret, nil
}

func (p *AuthHeadersClusterProvider) GetDerivedKubernetes(ctx context.Context, target string) (*kmskubernetes.Kubernetes, error) {
	authData, ok := ctx.Value(kmskubernetes.OAuthAuthorizationHeader).(string)
	if !ok {
		return nil, errors.New("authHeaders required")
	}

	authHeaders, err := NewK8sAuthHeadersFromString(authData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse auth headers: %w", err)
	}

	return NewKubernetes(authHeaders, p.config)
}

func (p *AuthHeadersClusterProvider) IsOpenShift(ctx context.Context) bool {
	klog.V(1).Infof("IsOpenShift not supported for auth-headers provider. Returning false.")
	return false
}

func (p *AuthHeadersClusterProvider) VerifyToken(ctx context.Context, target, token, audience string) (*authenticationv1api.UserInfo, []string, error) {
	return nil, nil, fmt.Errorf("VerifyToken not supported for auth-headers provider")
}

func (p *AuthHeadersClusterProvider) GetTargets(_ context.Context) ([]string, error) {
	klog.V(1).Infof("GetTargets not supported for auth-headers provider. Returning empty list.")
	return []string{""}, nil
}

func (p *AuthHeadersClusterProvider) GetTargetParameterName() string {
	klog.V(1).Infof("GetTargetParameterName not supported for auth-headers provider. Returning empty name.")
	return ""
}

func (p *AuthHeadersClusterProvider) GetDefaultTarget() string {
	klog.V(1).Infof("GetDefaultTarget not supported for auth-headers provider. Returning empty name.")
	return ""
}

func (p *AuthHeadersClusterProvider) WatchTargets(reload kmskubernetes.McpReload) {
	klog.V(1).Infof("WatchTargets not supported for auth-headers provider. Ignoring watch function.")
}

func (p *AuthHeadersClusterProvider) reset() error {
	klog.V(1).Infof("reset not supported for auth-headers provider. Ignoring reset function.")
	return nil
}

func (p *AuthHeadersClusterProvider) Close() {
}
