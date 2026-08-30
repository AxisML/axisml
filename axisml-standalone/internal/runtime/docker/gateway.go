package docker

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"

	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/yaml"
)

const (
	gatewayAPIVersion        = "gateway.networking.k8s.io/v1"
	envoyGatewayAPIVersion   = "gateway.envoyproxy.io/v1alpha1"
	gatewayParentName        = "axisml-gateway"
	gatewayBackendGroup      = "gateway.envoyproxy.io"
	gatewayServiceAnnotation = "standalone.axisml.io/service-name"
	gatewayDNSDomain         = ".axisml.local"
)

type gatewayResource struct {
	APIVersion string                `json:"apiVersion"`
	Kind       string                `json:"kind"`
	Metadata   gatewayObjectMetadata `json:"metadata"`
	Spec       any                   `json:"spec"`
}

type gatewayObjectMetadata struct {
	Name        string            `json:"name"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

type gatewayBackendSpec struct {
	Endpoints []gatewayEndpoint `json:"endpoints"`
}

type gatewayEndpoint struct {
	FQDN *gatewayFQDNEndpoint `json:"fqdn,omitempty"`
	IP   *gatewayIPEndpoint   `json:"ip,omitempty"`
}

type gatewayFQDNEndpoint struct {
	Hostname string `json:"hostname"`
	Port     int    `json:"port"`
}

type gatewayIPEndpoint struct {
	Address string `json:"address"`
	Port    int    `json:"port"`
}

type gatewayHTTPRouteSpec struct {
	ParentRefs []gatewayParentRef `json:"parentRefs"`
	Hostnames  []string           `json:"hostnames,omitempty"`
	Rules      []gatewayRouteRule `json:"rules"`
}

type gatewayParentRef struct {
	Name string `json:"name"`
}

type gatewayRouteRule struct {
	Matches     []gatewayRouteMatch `json:"matches"`
	BackendRefs []gatewayBackendRef `json:"backendRefs"`
}

type gatewayRouteMatch struct {
	Path gatewayPathMatch `json:"path"`
}

type gatewayPathMatch struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type gatewayBackendRef struct {
	Group  string `json:"group"`
	Kind   string `json:"kind"`
	Name   string `json:"name"`
	Weight *int32 `json:"weight,omitempty"`
}

// gatewayFileDocument is the readback shape used for status projection. It is
// deliberately the union of the Backend and HTTPRoute fields AxisML emits.
type gatewayFileDocument struct {
	APIVersion string                `json:"apiVersion"`
	Kind       string                `json:"kind"`
	Metadata   gatewayObjectMetadata `json:"metadata"`
	Spec       struct {
		Hostnames []string `json:"hostnames"`
		Rules     []struct {
			Matches []struct {
				Path gatewayPathMatch `json:"path"`
			} `json:"matches"`
			BackendRefs []gatewayBackendRef `json:"backendRefs"`
		} `json:"rules"`
	} `json:"spec"`
}

func gatewayBackend(name, serviceName string, endpoints []gatewayEndpoint) gatewayResource {
	if len(endpoints) == 0 {
		// Envoy Gateway's Backend CRD requires at least one endpoint. Keep the
		// resource valid while a service has no running replicas by pointing at
		// a closed loopback port inside the data-plane proxy. Readiness is still
		// projected from the real service containers, so this only produces the
		// expected upstream-unavailable response instead of routing elsewhere.
		endpoints = []gatewayEndpoint{{IP: &gatewayIPEndpoint{
			Address: "127.0.0.1",
			Port:    1,
		}}}
	}
	return gatewayResource{
		APIVersion: envoyGatewayAPIVersion,
		Kind:       "Backend",
		Metadata: gatewayObjectMetadata{
			Name:        name,
			Annotations: map[string]string{gatewayServiceAnnotation: serviceName},
		},
		Spec: gatewayBackendSpec{Endpoints: endpoints},
	}
}

func gatewayHTTPRoute(name, hostname, path string, refs []gatewayBackendRef) gatewayResource {
	if path == "" {
		path = "/"
	}
	spec := gatewayHTTPRouteSpec{
		ParentRefs: []gatewayParentRef{{Name: gatewayParentName}},
		Rules: []gatewayRouteRule{{
			Matches:     []gatewayRouteMatch{{Path: gatewayPathMatch{Type: "PathPrefix", Value: path}}},
			BackendRefs: refs,
		}},
	}
	if hostname != "" {
		spec.Hostnames = []string{hostname}
	}
	return gatewayResource{
		APIVersion: gatewayAPIVersion,
		Kind:       "HTTPRoute",
		Metadata:   gatewayObjectMetadata{Name: name},
		Spec:       spec,
	}
}

func gatewayBackendHostname(containerName string) string {
	return containerName + gatewayDNSDomain
}

func gatewayContainerEndpoint(containerName string, port int) gatewayEndpoint {
	return gatewayEndpoint{FQDN: &gatewayFQDNEndpoint{
		Hostname: gatewayBackendHostname(containerName),
		Port:     port,
	}}
}

func marshalGatewayResources(resources ...gatewayResource) ([]byte, error) {
	var out bytes.Buffer
	for i := range resources {
		b, err := yaml.Marshal(resources[i])
		if err != nil {
			return nil, err
		}
		if i > 0 {
			out.WriteString("---\n")
		}
		out.Write(b)
	}
	return out.Bytes(), nil
}

func readGatewayDocuments(path string) ([]gatewayFileDocument, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	decoder := yamlutil.NewYAMLOrJSONDecoder(bytes.NewReader(b), 4096)
	var docs []gatewayFileDocument
	for {
		var doc gatewayFileDocument
		if err := decoder.Decode(&doc); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		if doc.Kind != "" {
			docs = append(docs, doc)
		}
	}
	return docs, nil
}

func gatewayRouteEndpoint(doc gatewayFileDocument) string {
	path := "/"
	if len(doc.Spec.Rules) > 0 && len(doc.Spec.Rules[0].Matches) > 0 &&
		doc.Spec.Rules[0].Matches[0].Path.Value != "" {
		path = doc.Spec.Rules[0].Matches[0].Path.Value
	}
	if len(doc.Spec.Hostnames) > 0 && doc.Spec.Hostnames[0] != "" {
		return "http://" + doc.Spec.Hostnames[0] + path
	}
	return path
}

func (r *Runtime) writeGatewayFile(path string, b []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	// Envoy Gateway watches every file directly under GatewayConfigDir. Build
	// temporary files in a non-recursive child directory, then atomically rename
	// them into place so the file provider never observes a partial manifest.
	tmpDir := filepath.Join(dir, ".tmp")
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(tmpDir, filepath.Base(path)+"-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if _, err := tmp.Write(b); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o644); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("install gateway resources: %w", err)
	}
	return nil
}
