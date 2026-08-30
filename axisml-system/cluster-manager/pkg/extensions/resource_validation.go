package extensions

import corev1 "k8s.io/api/core/v1"

// ResourceListValidator validates one scheduler-facing resource map at an API
// mutation boundary. Composition roots may supply a deployment-form-specific
// validator; nil selects Cluster Manager's Kubernetes-compatible default.
type ResourceListValidator func(field string, resources corev1.ResourceList) error
