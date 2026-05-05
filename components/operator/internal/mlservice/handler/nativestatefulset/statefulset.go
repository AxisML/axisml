package nativestatefulset

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	axisml "github.com/axisml/axisml/components/operator/api/mlservice/v1alpha1"
)

const (
	// modelEnvVarName carries the resolved Artifacts model URI. MVP synthesises
	// a model://<name>:<version> placeholder identical to the deployment
	// handler — real Artifacts resolution arrives with the client rewrite.
	modelEnvVarName = "AXISML_MODEL_URI"

	// replicaIndexEnvVarName surfaces the K8s-injected
	// apps.kubernetes.io/pod-index label as an env var (§6.6.2). Static pod
	// labels can't reference fieldRef, so materialising axisml.io/replica-index
	// per ordinal would require a mutating webhook; the downward-API env var
	// is the webhook-free MVP fit.
	replicaIndexEnvVarName = "AXISML_REPLICA_INDEX"
	replicaIndexFieldPath  = "metadata.labels['apps.kubernetes.io/pod-index']"
)

// buildStatefulSet renders the desired StatefulSet for the (native, statefulset)
// backend. Pod template injection follows §3.4 Pod 注入约定.
func buildStatefulSet(mls *axisml.MLService, cfg Config) *appsv1.StatefulSet {
	role := mls.Spec.Roles[0]
	selector := selectorLabels(mls, role.Name)
	podLabels := podTemplateLabels(mls, role.Name)
	replicas := role.Replicas

	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mls.Name,
			Namespace: mls.Namespace,
			Labels:    resourceLabels(mls, role.Name),
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas:            &replicas,
			Selector:            &metav1.LabelSelector{MatchLabels: selector},
			ServiceName:         defaultedServiceName(mls, cfg),
			PodManagementPolicy: defaultedPodManagementPolicy(cfg),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: podLabels},
				Spec: corev1.PodSpec{
					SchedulerName:     axisml.SchedulerName,
					NodeSelector:      mls.Spec.Scheduling.NodeSelector,
					Tolerations:       mls.Spec.Scheduling.Tolerations,
					PriorityClassName: mls.Spec.Scheduling.PriorityClass,
					Containers:        []corev1.Container{buildContainer(mls, role)},
				},
			},
		},
	}
	return sts
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
	}
	c.Env = append(c.Env, corev1.EnvVar{
		Name:  modelEnvVarName,
		Value: fmt.Sprintf("model://%s:%s", mls.Spec.ModelRef.Name, mls.Spec.ModelRef.Version),
	})
	c.Env = append(c.Env, corev1.EnvVar{
		Name: replicaIndexEnvVarName,
		ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				APIVersion: "v1",
				FieldPath:  replicaIndexFieldPath,
			},
		},
	})
	for _, p := range tmpl.Ports {
		c.Ports = append(c.Ports, corev1.ContainerPort{
			Name:          p.Name,
			ContainerPort: p.ContainerPort,
			Protocol:      p.Protocol,
		})
	}
	return c
}

// selectorLabels are the Pod labels used to match the StatefulSet / headless
// Service selector. They MUST stay stable across updates — these are the §3.4
// labels minus role-internal observability bits (axisml.io/quota,
// quota.scheduling.koordinator.sh/name).
func selectorLabels(mls *axisml.MLService, role string) map[string]string {
	return map[string]string{
		axisml.LabelServiceID: mls.Labels[axisml.LabelServiceID],
		axisml.LabelRole:      role,
	}
}

// podTemplateLabels include selector keys plus §3.4 quota / tenant labels.
// Quota / tenant evolve over the CR's lifetime and must NOT be in the
// selector.
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

// resourceLabels are written onto the StatefulSet / Service / HTTPRoute
// objects themselves so operators can list-by-label across resource types.
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
