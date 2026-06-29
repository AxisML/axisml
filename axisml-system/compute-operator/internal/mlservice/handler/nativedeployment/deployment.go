package nativedeployment

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	axisml "github.com/axisml/axisml/axisml-system/compute-operator/api/mlservice/v1alpha1"
)

// buildDeployment renders the desired Deployment for the (native, deployment)
// backend. Pod template injection follows §6 Pod 注入约定.
func buildDeployment(mls *axisml.MLService) *appsv1.Deployment {
	role := mls.Spec.Roles[0]
	selector := selectorLabels(mls, role.Name)
	podLabels := podTemplateLabels(mls, role.Name)
	replicas := role.Replicas

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mls.Name,
			Namespace: mls.Namespace,
			Labels:    resourceLabels(mls, role.Name),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: selector},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: podLabels},
				Spec: corev1.PodSpec{
					SchedulerName:     axisml.SchedulerName,
					NodeSelector:      mls.Spec.Scheduling.NodeSelector,
					Tolerations:       mls.Spec.Scheduling.Tolerations,
					PriorityClassName: mls.Spec.Scheduling.PriorityClass,
					Volumes:           role.Template.Volumes,
					Containers:        []corev1.Container{buildContainer(mls, role)},
				},
			},
		},
	}
	if pds := mls.Spec.RunPolicy.ProgressDeadlineSeconds; pds != nil {
		dep.Spec.ProgressDeadlineSeconds = pds
	}
	return dep
}

func buildContainer(mls *axisml.MLService, role axisml.RoleSpec) corev1.Container {
	tmpl := role.Template
	c := corev1.Container{
		Name:            role.Name,
		Image:           tmpl.Image,
		ImagePullPolicy: tmpl.ImagePullPolicy,
		Command:         tmpl.Command,
		Args:            tmpl.Args,
		WorkingDir:      tmpl.WorkingDir,
		Env:             append([]corev1.EnvVar(nil), tmpl.Env...),
		EnvFrom:         tmpl.EnvFrom,
		Resources:       tmpl.Resources,
		VolumeMounts:    tmpl.VolumeMounts,
	}
	for _, p := range tmpl.Ports {
		c.Ports = append(c.Ports, corev1.ContainerPort{
			Name:          p.Name,
			ContainerPort: p.ContainerPort,
			Protocol:      p.Protocol,
		})
	}
	return c
}

// selectorLabels are the Pod labels used to match a Service / Deployment
// selector. They MUST stay stable across updates — these are the §6 labels
// minus role-internal observability bits (e.g. axisml.io/quota/koord-quota).
func selectorLabels(mls *axisml.MLService, role string) map[string]string {
	return map[string]string{
		axisml.LabelServiceID: mls.Labels[axisml.LabelServiceID],
		axisml.LabelRole:      role,
	}
}

// podTemplateLabels are written onto the Pod template. They include the
// selector keys plus the §6 Pod 注入约定 labels (Koord quota, axisml quota,
// tenant). These are NOT part of the selector — that lets the quota label
// evolve without triggering Service-selector mismatches.
func podTemplateLabels(mls *axisml.MLService, role string) map[string]string {
	out := selectorLabels(mls, role)
	out[axisml.LabelKoordQuotaName] = mls.Spec.Scheduling.Quota
	if v := mls.Labels[axisml.LabelQuota]; v != "" {
		out[axisml.LabelQuota] = v
	}
	if v := mls.Labels[axisml.LabelTenant]; v != "" {
		out[axisml.LabelTenant] = v
	}
	return out
}

// resourceLabels are written onto the Deployment / Service / HTTPRoute
// objects themselves. axisml.io/service-id is always present so operators can
// list-by-label across resource types.
func resourceLabels(mls *axisml.MLService, role string) map[string]string {
	out := map[string]string{
		axisml.LabelServiceID: mls.Labels[axisml.LabelServiceID],
		axisml.LabelRole:      role,
	}
	if v := mls.Labels[axisml.LabelTenant]; v != "" {
		out[axisml.LabelTenant] = v
	}
	if v := mls.Labels[axisml.LabelQuota]; v != "" {
		out[axisml.LabelQuota] = v
	}
	return out
}
