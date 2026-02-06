package kubernetes

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	kmskubernetes "github.com/containers/kubernetes-mcp-server/pkg/kubernetes"
)

// AuthType represents the type of Kubernetes authentication.
type AuthType string
type ContextKey string

const (
	// CustomServerHeader is the Kubernetes cluster URL.
	CustomServerHeader = kmskubernetes.HeaderKey("x-target-k8s-server")
	// CustomCertificateAuthorityData is the base64-encoded CA certificate.
	CustomCertificateAuthorityDataHeader = kmskubernetes.HeaderKey("x-target-k8s-certificate-authority-data")
	// CustomAuthorizationHeader is the optional bearer token for authentication.
	CustomAuthorizationHeader = kmskubernetes.HeaderKey("x-target-k8s-authorization")

	// CustomClientCertificateData is the base64-encoded client certificate.
	CustomClientCertificateDataHeader = kmskubernetes.HeaderKey("x-target-k8s-client-certificate-data")
	// CustomClientKeyData is the base64-encoded client key.
	CustomClientKeyDataHeader = kmskubernetes.HeaderKey("x-target-k8s-client-key-data")
	// CustomInsecureSkipTLSVerify is the optional flag to skip TLS verification.
	CustomInsecureSkipTLSVerifyHeader = kmskubernetes.HeaderKey("x-target-k8s-insecure-skip-tls-verify")
)

// K8sAuthHeaders represents Kubernetes API authentication headers.
type K8sAuthHeaders struct {
	// Server is the Kubernetes cluster URL.
	Server string
	// ClusterCertificateAuthorityData is the Certificate Authority data.
	CertificateAuthorityData []byte
	// AuthorizationToken is the optional bearer token for authentication.
	AuthorizationToken string
	// ClientCertificateData is the optional client certificate data.
	ClientCertificateData []byte
	// ClientKeyData is the optional client key data.
	ClientKeyData []byte
	// InsecureSkipTLSVerify is the optional flag to skip TLS verification.
	InsecureSkipTLSVerify bool
}

// GetDecodedData decodes and returns the data.
func GetDecodedData(data string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(data)
}

func NewK8sAuthHeadersFromString(data string) (*K8sAuthHeaders, error) {
	var ok bool
	var err error

	payload := []byte(data)
	// check if data is base64-encoded string
	decodedData, err := base64.StdEncoding.DecodeString(data)
	if err == nil {
		payload = decodedData
	}

	// Unmarshal the decoded data into a map.
	var authDataMap map[string]any
	err = json.Unmarshal([]byte(payload), &authDataMap)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal auth data: %w", err)
	}

	// convert keys to lower case to make header keys case-insensitive
	authDataMapLower := make(map[string]any)
	for k, v := range authDataMap {
		authDataMapLower[strings.ToLower(k)] = v
	}
	authDataMap = authDataMapLower

	// Initialize auth headers with default values.
	authHeaders := &K8sAuthHeaders{
		InsecureSkipTLSVerify: false,
	}

	// Get cluster URL from headers.
	authHeaders.Server, ok = authDataMap[string(CustomServerHeader)].(string)
	if !ok || authHeaders.Server == "" {
		return nil, fmt.Errorf("%s header is required", CustomServerHeader)
	}

	// Get certificate authority data from headers.
	certificateAuthorityDataBase64, ok := authDataMap[string(CustomCertificateAuthorityDataHeader)].(string)
	if !ok || certificateAuthorityDataBase64 == "" {
		return nil, fmt.Errorf("%s header is required", CustomCertificateAuthorityDataHeader)
	}
	// Decode certificate authority data.
	authHeaders.CertificateAuthorityData, err = GetDecodedData(certificateAuthorityDataBase64)
	if err != nil {
		return nil, fmt.Errorf("invalid certificate authority data: %w", err)
	}

	// Get insecure skip TLS verify flag from headers.
	if authDataMap[string(CustomInsecureSkipTLSVerifyHeader)] != nil && strings.ToLower(authDataMap[string(CustomInsecureSkipTLSVerifyHeader)].(string)) == "true" {
		authHeaders.InsecureSkipTLSVerify = true
	}

	// Get authorization token from headers.
	authHeaders.AuthorizationToken, _ = authDataMap[string(CustomAuthorizationHeader)].(string)

	// Get client certificate data from headers.
	clientCertificateDataBase64, _ := authDataMap[string(CustomClientCertificateDataHeader)].(string)
	if clientCertificateDataBase64 != "" {
		authHeaders.ClientCertificateData, err = GetDecodedData(clientCertificateDataBase64)
		if err != nil {
			return nil, fmt.Errorf("invalid client certificate data: %w", err)
		}
	}
	// Get client key data from headers.
	clientKeyDataBase64, _ := authDataMap[string(CustomClientKeyDataHeader)].(string)
	if clientKeyDataBase64 != "" {
		authHeaders.ClientKeyData, err = GetDecodedData(clientKeyDataBase64)
		if err != nil {
			return nil, fmt.Errorf("invalid client key data: %w", err)
		}
	}

	// Check if a valid authentication type is provided.
	if !authHeaders.IsValid() {
		return nil, fmt.Errorf("either %s header for token authentication or (%s and %s) headers for client certificate authentication required", CustomAuthorizationHeader, CustomClientCertificateDataHeader, CustomClientKeyDataHeader)
	}

	return authHeaders, nil
}

// IsValid checks if the authentication headers are valid.
func (h *K8sAuthHeaders) IsValid() bool {
	if h.AuthorizationToken != "" {
		return true
	}
	if len(h.ClientCertificateData) > 0 && len(h.ClientKeyData) > 0 {
		return true
	}
	return false
}
