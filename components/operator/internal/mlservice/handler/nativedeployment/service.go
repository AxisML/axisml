package nativedeployment

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	axisml "github.com/axisml/axisml/components/operator/api/mlservice/v1alpha1"
)

// buildService renders the ClusterIP Service for the predictor role.
// Each declared role port becomes a Service port with targetPort=containerPort
// (§8.1). Service name == MLService name keeps the in-cluster DNS predictable.
func buildService(mls *axisml.MLService) *corev1.Service {
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
			Name:      mls.Name,
			Namespace: mls.Namespace,
			Labels:    resourceLabels(mls, role.Name),
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP,
			Selector: selectorLabels(mls, role.Name),
			Ports:    ports,
		},
	}
}
