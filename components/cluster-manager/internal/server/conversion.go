package server

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	axismlv1alpha1 "github.com/axisml/axisml/components/cluster-manager/api/v1alpha1"
)

// PoolToDTO renders a ResourcePool CR into its REST DTO.
func PoolToDTO(p *axismlv1alpha1.ResourcePool) ResourcePoolDTO {
	dto := ResourcePoolDTO{
		Name:            p.Name,
		Description:     p.Annotations[DescriptionAnnotation],
		NodeSelector:    p.Spec.NodeSelector,
		Tolerations:     p.Spec.Tolerations,
		Units:           make([]ResourceUnitDTO, 0, len(p.Spec.Units)),
		Labels:          p.Labels,
		Annotations:     stripReservedAnnotations(p.Annotations),
		ResourceVersion: p.ResourceVersion,
		CreatedAt:       p.CreationTimestamp.Time,
	}
	for _, u := range p.Spec.Units {
		dto.Units = append(dto.Units, UnitToDTO(u))
	}
	return dto
}

// UnitToDTO renders a ResourceUnit embedded entry into its REST DTO.
func UnitToDTO(u axismlv1alpha1.ResourceUnit) ResourceUnitDTO {
	return ResourceUnitDTO{
		Name:         u.Name,
		Description:  u.Annotations[DescriptionAnnotation],
		Requests:     u.Requests,
		Limits:       u.Limits,
		NodeSelector: u.NodeSelector,
		Annotations:  stripReservedAnnotations(u.Annotations),
	}
}

// DTOToPool maps a create request into a fresh ResourcePool CR. The caller
// is responsible for ObjectMeta.ResourceVersion / Annotations[user].
func DTOToPool(req CreateResourcePoolRequest, lastModifiedBy string) *axismlv1alpha1.ResourcePool {
	pool := &axismlv1alpha1.ResourcePool{
		ObjectMeta: metav1.ObjectMeta{
			Name:        req.Name,
			Labels:      copyMap(req.Labels),
			Annotations: mergeAnnotations(req.Annotations, req.Description, lastModifiedBy),
		},
		Spec: axismlv1alpha1.ResourcePoolSpec{
			NodeSelector: copyMap(req.NodeSelector),
			Tolerations:  copyTolerations(req.Tolerations),
		},
	}
	for _, u := range req.Units {
		pool.Spec.Units = append(pool.Spec.Units, axismlv1alpha1.ResourceUnit{
			Name:         u.Name,
			Requests:     u.Requests,
			Limits:       u.Limits,
			NodeSelector: copyMap(u.NodeSelector),
			Annotations:  mergeAnnotations(u.Annotations, u.Description, ""),
		})
	}
	return pool
}

// DTOToUnit converts a unit create request into the embedded array entry.
func DTOToUnit(req CreateResourceUnitRequest) axismlv1alpha1.ResourceUnit {
	return axismlv1alpha1.ResourceUnit{
		Name:         req.Name,
		Requests:     req.Requests,
		Limits:       req.Limits,
		NodeSelector: copyMap(req.NodeSelector),
		Annotations:  mergeAnnotations(req.Annotations, req.Description, ""),
	}
}

func mergeAnnotations(user map[string]string, description, lastModifiedBy string) map[string]string {
	out := copyMap(user)
	if out == nil {
		out = map[string]string{}
	}
	if description != "" {
		out[DescriptionAnnotation] = description
	}
	if lastModifiedBy != "" {
		out[LastModifiedByAnnotation] = lastModifiedBy
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func stripReservedAnnotations(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		if k == DescriptionAnnotation || k == LastModifiedByAnnotation {
			continue
		}
		out[k] = v
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func copyMap(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func copyTolerations(in []corev1.Toleration) []corev1.Toleration {
	if in == nil {
		return nil
	}
	out := make([]corev1.Toleration, len(in))
	for i := range in {
		in[i].DeepCopyInto(&out[i])
	}
	return out
}
