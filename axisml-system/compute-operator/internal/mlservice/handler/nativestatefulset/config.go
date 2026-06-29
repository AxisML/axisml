package nativestatefulset

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation"

	axisml "github.com/axisml/axisml/axisml-system/compute-operator/api/mlservice/v1alpha1"
)

// Config is the parsed shape of MLService.spec.backend.config for the
// (native, statefulset) backend. Per §6.6.2, only podManagementPolicy and
// serviceName are recognized at MVP; everything else (volumeClaimTemplates,
// updateStrategy, ...) is rejected by parseBackendConfig so a future field
// cannot silently no-op while the handler does not yet support it.
type Config struct {
	PodManagementPolicy appsv1.PodManagementPolicyType `json:"podManagementPolicy,omitempty"`
	ServiceName         string                         `json:"serviceName,omitempty"`
}

// parseBackendConfig strict-decodes the user-supplied JSON. Empty / nil input
// returns a zero Config. Unknown keys, invalid podManagementPolicy values,
// and non-DNS-1123 serviceName values fail fast.
func parseBackendConfig(raw *runtime.RawExtension) (Config, error) {
	if raw == nil || len(raw.Raw) == 0 {
		return Config{}, nil
	}
	dec := json.NewDecoder(bytes.NewReader(raw.Raw))
	dec.DisallowUnknownFields()
	var c Config
	if err := dec.Decode(&c); err != nil {
		return Config{}, err
	}
	switch c.PodManagementPolicy {
	case "", appsv1.OrderedReadyPodManagement, appsv1.ParallelPodManagement:
	default:
		return Config{}, fmt.Errorf("podManagementPolicy %q must be OrderedReady or Parallel",
			c.PodManagementPolicy)
	}
	if c.ServiceName != "" {
		if errs := validation.IsDNS1123Label(c.ServiceName); len(errs) > 0 {
			return Config{}, fmt.Errorf("serviceName %q invalid: %s",
				c.ServiceName, strings.Join(errs, "; "))
		}
	}
	return c, nil
}

// defaultedServiceName returns the headless Service name. When the user does
// not override it via backend.config.serviceName, the MLService's own name
// is used so in-cluster DNS stays predictable.
func defaultedServiceName(mls *axisml.MLService, c Config) string {
	if c.ServiceName != "" {
		return c.ServiceName
	}
	return mls.Name
}

// defaultedPodManagementPolicy applies §6.6.2's OrderedReady default when the
// user leaves backend.config.podManagementPolicy unset.
func defaultedPodManagementPolicy(c Config) appsv1.PodManagementPolicyType {
	if c.PodManagementPolicy == "" {
		return appsv1.OrderedReadyPodManagement
	}
	return c.PodManagementPolicy
}
