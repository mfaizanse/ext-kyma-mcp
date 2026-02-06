package kubernetes

import (
	kmsapi "github.com/containers/kubernetes-mcp-server/pkg/api"
	kmskubernetes "github.com/containers/kubernetes-mcp-server/pkg/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func NewKubernetes(authHeaders *K8sAuthHeaders, config kmsapi.BaseConfig) (*kmskubernetes.Kubernetes, error) {

	var certData []byte = nil
	if len(authHeaders.ClientCertificateData) > 0 {
		certData = authHeaders.ClientCertificateData
	}

	var keyData []byte = nil
	if len(authHeaders.ClientKeyData) > 0 {
		keyData = authHeaders.ClientKeyData
	}

	restConfig := &rest.Config{
		Host:        authHeaders.Server,
		BearerToken: authHeaders.AuthorizationToken,
		TLSClientConfig: rest.TLSClientConfig{
			Insecure: authHeaders.InsecureSkipTLSVerify,
			CAData:   authHeaders.CertificateAuthorityData,
			CertData: certData,
			KeyData:  keyData,
		},
	}
	// Create a dummy kubeconfig clientcmdapi.Config to be used in places where clientcmd.ClientConfig is required.
	clientCmdConfig := clientcmdapi.NewConfig()
	clientCmdConfig.Clusters["cluster"] = &clientcmdapi.Cluster{
		Server:                authHeaders.Server,
		InsecureSkipTLSVerify: authHeaders.InsecureSkipTLSVerify,
	}
	clientCmdConfig.AuthInfos["user"] = &clientcmdapi.AuthInfo{
		Token:                 authHeaders.AuthorizationToken,
		ClientCertificateData: certData,
		ClientKeyData:         keyData,
	}

	return kmskubernetes.NewKubernetes(config, clientcmd.NewDefaultClientConfig(*clientCmdConfig, nil), restConfig)
}
