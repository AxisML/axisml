// Package workloadconfig contains configuration objects shared by MLRun and
// MLService specs.
package workloadconfig

// ConfigMap declares a workload-owned Kubernetes ConfigMap. The compute
// operator creates it in the workload namespace before reconciling the
// underlying Job, Deployment, or StatefulSet. Workload pod templates reference
// it by Name through configMapKeyRef, envFrom.configMapRef, or volumes.configMap.
type ConfigMap struct {
	// Name is the ConfigMap name in the workload namespace.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	Name string `json:"name" binding:"required,configmap_name" desc:"DNS-1123 name of the workload-owned ConfigMap created in the workload namespace."`
	// Data is written to the Kubernetes ConfigMap data field.
	// +optional
	Data map[string]string `json:"data,omitempty" desc:"UTF-8 configuration entries keyed by file or environment-variable name."`
}
