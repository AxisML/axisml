package nativestatefulset

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	axisml "github.com/axisml/axisml/axisml-system/apis/mlservice/v1alpha1"
)

// buildHeadlessService renders the headless Service that fronts the
// StatefulSet. ClusterIP=None gives every Pod a stable per-replica DNS name
// of the form <pod>.<svc>.<namespace>.svc.cluster.local (§6.6.2).
//
// publishNotReadyAddresses is intentionally left unset — leader-election
// peers that need it can opt in via a future backend.config field once a
// concrete consumer materialises.
func buildHeadlessService(mls *axisml.MLService, cfg Config) *corev1.Service {
	role := mls.Spec.Roles[0]
	ports := make([]corev1.ServicePort, 0, len(role.Template.Ports))
	for _, p := range role.Template.Ports {
		protocol := p.Protocol
		if protocol == "" {
			protocol = corev1.ProtocolTCP
		}
		ports = append(ports, corev1.ServicePort{
			Name:       p.Name,
			Port:       p.ContainerPort,
			TargetPort: intstr.FromInt(int(p.ContainerPort)),
			Protocol:   protocol,
		})
	}
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaultedServiceName(mls, cfg),
			Namespace: mls.Namespace,
			Labels:    resourceLabels(mls, role.Name),
		},
		Spec: corev1.ServiceSpec{
			Type:      corev1.ServiceTypeClusterIP,
			ClusterIP: corev1.ClusterIPNone,
			Selector:  selectorLabels(mls, role.Name),
			Ports:     ports,
		},
	}
}
