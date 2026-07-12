package standalone

import (
	"context"
	"fmt"
	"io"

	"github.com/docker/docker/api/types/container"
	corev1 "k8s.io/api/core/v1"
	eventsv1 "k8s.io/api/events/v1"
	"k8s.io/apimachinery/pkg/types"

	mlservicev1alpha1 "github.com/axisml/axisml/axisml-system/apis/mlservice/v1alpha1"
)

// renderServicePlans renders a (native, deployment|statefulset) MLService into
// one ContainerPlan per replica of its single role. Unsupported backends and
// fields surface as CapabilityError.
func (r *Runtime) renderServicePlans(svc *mlservicev1alpha1.MLService) ([]ContainerPlan, error) {
	if svc.Spec.Backend.Name != "native" ||
		(svc.Spec.Backend.Engine != "deployment" && svc.Spec.Backend.Engine != "statefulset") {
		return nil, capabilityError("MLService backend %s/%s is unsupported in Lite (only native/deployment|statefulset)",
			svc.Spec.Backend.Name, svc.Spec.Backend.Engine)
	}
	if len(svc.Spec.Roles) == 0 {
		return nil, capabilityError("MLService %s/%s has no roles", svc.Namespace, svc.Name)
	}
	role := svc.Spec.Roles[0]
	tmpl := role.Template
	if len(tmpl.EnvFrom) > 0 {
		return nil, capabilityError("MLService envFrom is unsupported in Lite")
	}
	env, err := envToStrings(tmpl.Env)
	if err != nil {
		return nil, err
	}
	ns, name := svc.Namespace, svc.Name

	var ports []PortPlan
	for _, p := range tmpl.Ports {
		ports = append(ports, PortPlan{Name: p.Name, ContainerPort: p.ContainerPort, Protocol: protoString(p.Protocol)})
	}
	mounts, err := r.volumeMounts(ns, name, tmpl.Volumes, tmpl.VolumeMounts)
	if err != nil {
		return nil, err
	}

	var plans []ContainerPlan
	for i := 0; i < int(role.Replicas); i++ {
		labels := r.baseLabels(KindService, ns, name)
		labels[LabelRole] = roleOr(role.Name)
		labels[LabelReplicaIndex] = formatLabelInt(i)
		p := ContainerPlan{
			Name:          r.containerName(KindService, ns, name, roleOr(role.Name), i),
			Image:         tmpl.Image,
			Command:       tmpl.Command,
			Args:          tmpl.Args,
			Env:           env,
			WorkingDir:    tmpl.WorkingDir,
			Labels:        labels,
			Ports:         ports,
			Mounts:        mounts,
			Resources:     resourcePlan(tmpl.Resources),
			RestartPolicy: string(container.RestartPolicyUnlessStopped),
		}
		p.Labels[LabelSpecHash] = specHash(planIdentity(&p))
		plans = append(plans, p)
	}
	return plans, nil
}

// volumeMounts maps each declared volumeMount onto a Docker mount. It is shared
// by MLService and MLRun rendering, so a training run mounts a dataset volume by
// exactly the same rules a service mounts its workspace. Resolution order per
// mount:
//   - a predefined host-path volume (claim name registered in cfg.HostPathVolumes)
//     → a bind mount to the host directory (Lite-only "hostPath" support);
//   - a PVC-backed volume → the managed Docker volume keyed on its claim name —
//     the same name the VolumeManager (Runtime.Ensure / Runtime.Delete)
//     materialises and reclaims, so mount, provision and retention target agree;
//   - any other declared volume → a per-(namespace, name, volume) managed volume.
func (r *Runtime) volumeMounts(namespace, name string, volumes []corev1.Volume, volumeMounts []corev1.VolumeMount) ([]MountPlan, error) {
	if len(volumeMounts) == 0 {
		return nil, nil
	}
	declared := map[string]corev1.Volume{}
	for _, v := range volumes {
		declared[v.Name] = v
	}
	var mounts []MountPlan
	for _, vm := range volumeMounts {
		vol, ok := declared[vm.Name]
		if !ok {
			return nil, capabilityError("volumeMount %q references an undeclared volume", vm.Name)
		}
		if hp := r.hostPathFor(vol); hp != "" {
			mounts = append(mounts, MountPlan{
				Type:     "bind",
				Source:   hp,
				Target:   vm.MountPath,
				ReadOnly: vm.ReadOnly,
			})
			continue
		}
		source := r.volumeName(namespace, name, vm.Name)
		if vol.PersistentVolumeClaim != nil {
			source = r.pvcVolumeName(namespace, vol.PersistentVolumeClaim.ClaimName)
		}
		mounts = append(mounts, MountPlan{
			Type:     "volume",
			Source:   source,
			Target:   vm.MountPath,
			ReadOnly: vm.ReadOnly,
		})
	}
	return mounts, nil
}

// hostPathFor returns the host directory to bind-mount for a declared volume, or
// "" when it is not host-backed. It resolves a PVC-backed volume whose claim name
// is registered as a predefined hostPath volume (cfg.HostPathVolumes) — so a
// workspace/run that references the volume by claim name (its ordinary PVC form)
// transparently binds the host path. Only admin-declared paths are honoured; a
// raw hostPath source in a pod template is not, keeping the bindable set to the
// tenant's predefined registry.
func (r *Runtime) hostPathFor(vol corev1.Volume) string {
	if vol.PersistentVolumeClaim == nil {
		return ""
	}
	return r.cfg.HostPathVolumes[vol.PersistentVolumeClaim.ClaimName]
}

func (r *Runtime) volumeName(namespace, name, vol string) string {
	raw := fmt.Sprintf("axisml-vol-%s-%s-%s", namespace, name, vol)
	clean := nameSanitizer.ReplaceAllString(raw, "-")
	if clean == raw && len(clean) <= 100 {
		return clean
	}
	return fmt.Sprintf("axisml-vol-%s", shortHash(raw))
}

// ApplyMLService idempotently converges the Service's replicas: it ensures the
// backing volumes, (re)creates changed/missing replicas and removes surplus
// ones (scale down).
func (r *Runtime) ApplyMLService(ctx context.Context, desired *mlservicev1alpha1.MLService) error {
	ns, name := desired.Namespace, desired.Name
	plans, err := r.renderServicePlans(desired)
	if err != nil {
		return err
	}
	existing, err := r.listContainers(ctx, KindService, ns, name)
	if err != nil {
		return err
	}
	byName := indexByName(existing)
	wanted := map[string]struct{}{}

	// Ensure backing volumes once.
	for i := range plans {
		for _, m := range plans[i].Mounts {
			if m.Type != "volume" {
				continue
			}
			vlabels := r.baseLabels(KindService, ns, name)
			if err := r.ensureVolume(ctx, m.Source, vlabels); err != nil {
				return fmt.Errorf("ensure volume %q: %w", m.Source, err)
			}
		}
	}

	// On a mid-loop create failure, already-created replicas are left running and
	// the error is returned; the reconcile loop re-applies idempotently (unchanged
	// replicas skip by spec hash, missing ones are created) and converges the
	// rest. Tearing created replicas back down would only reduce availability and
	// force a full rebuild next apply.
	for i := range plans {
		p := &plans[i]
		wanted[p.Name] = struct{}{}
		if cur, ok := byName[p.Name]; ok {
			if cur.Labels[LabelSpecHash] == p.Labels[LabelSpecHash] {
				continue
			}
			if err := r.removeContainer(ctx, cur.ID); err != nil {
				return err
			}
		}
		// Always attempt a pull; on failure fall back to a locally-present image.
		if err := r.pullImage(ctx, p.Image); err != nil {
			r.log.Info("image pull failed, trying local", "image", p.Image, "err", err.Error())
		}
		if _, err := r.createAndStart(ctx, p); err != nil {
			r.events.record(KindService, ns, name, "", "ApplyFailed", err.Error())
			return err
		}
	}

	// Scale down: remove replicas no longer desired.
	for _, c := range existing {
		if _, ok := wanted[summaryName(c)]; !ok {
			if err := r.removeContainer(ctx, c.ID); err != nil {
				return err
			}
		}
	}

	// Service-level routing: a spec.route exposes the service through Traefik
	// (the mechanism workspace / tensorboard and any routed service rely on).
	if err := r.applyServiceRoute(desired, plans); err != nil {
		r.events.record(KindService, ns, name, "", "RouteFailed", err.Error())
		return err
	}
	r.events.record(KindService, ns, name, "", "Applied", "service converged")
	return nil
}

// ObserveMLService aggregates the Service's container states into a status.
func (r *Runtime) ObserveMLService(ctx context.Context, key types.NamespacedName) (mlservicev1alpha1.MLServiceStatus, error) {
	conts, err := r.listContainers(ctx, KindService, key.Namespace, key.Name)
	if err != nil {
		return mlservicev1alpha1.MLServiceStatus{}, err
	}
	if len(conts) == 0 {
		return mlservicev1alpha1.MLServiceStatus{}, notFound("mlservices", key)
	}
	states, err := r.inspectAll(ctx, conts)
	if err != nil {
		return mlservicev1alpha1.MLServiceStatus{}, err
	}

	var ready, failed int
	for i := range states {
		switch {
		case states[i].ready():
			ready++
		case states[i].status == "exited" && states[i].exitCode != 0:
			failed++
		}
	}
	total := len(states)
	status := mlservicev1alpha1.MLServiceStatus{ReadyReplicas: int32(ready)}
	switch {
	case ready == total:
		status.Phase = mlservicev1alpha1.PhaseReady
	case ready > 0:
		status.Phase = mlservicev1alpha1.PhaseDegraded
	case failed == total:
		status.Phase = mlservicev1alpha1.PhaseFailed
		status.Message = "all replicas failed"
	default:
		status.Phase = mlservicev1alpha1.PhasePending
	}
	status.Endpoint = r.serviceRouteEndpoint(key.Namespace, key.Name)
	return status, nil
}

// DeleteMLService removes the Service's containers and backing volumes.
func (r *Runtime) DeleteMLService(ctx context.Context, key types.NamespacedName) error {
	conts, err := r.listContainers(ctx, KindService, key.Namespace, key.Name)
	if err != nil {
		return err
	}
	for _, c := range conts {
		if err := r.removeContainer(ctx, c.ID); err != nil {
			return err
		}
	}
	if err := r.deleteServiceRoute(key.Namespace, key.Name); err != nil {
		return err
	}
	// Volumes are NOT removed here: their lifecycle follows the retention policy
	// and is driven explicitly through the VolumeManager (Runtime.Delete), so
	// deleting a workspace never silently wipes user data the policy meant to
	// keep.
	if len(conts) > 0 {
		r.events.record(KindService, key.Namespace, key.Name, "", "Deleted", "service containers removed")
	}
	return nil
}

// ListMLServiceInstances projects the Service's containers as Pods.
func (r *Runtime) ListMLServiceInstances(ctx context.Context, key types.NamespacedName) (*corev1.PodList, error) {
	conts, err := r.listContainers(ctx, KindService, key.Namespace, key.Name)
	if err != nil {
		return nil, err
	}
	return r.projectPods(ctx, key.Namespace, conts)
}

// GetMLServiceInstanceLogs streams an instance's logs after verifying ownership.
func (r *Runtime) GetMLServiceInstanceLogs(ctx context.Context, key types.NamespacedName, instance string, opts *corev1.PodLogOptions) (io.ReadCloser, error) {
	if err := r.verifyInstance(ctx, KindService, key, instance); err != nil {
		return nil, err
	}
	return r.streamLogs(ctx, instance, opts)
}

// GetMLServiceInstanceEvents returns runtime events for an instance.
func (r *Runtime) GetMLServiceInstanceEvents(ctx context.Context, key types.NamespacedName, instance string) (*eventsv1.EventList, error) {
	if err := r.verifyInstance(ctx, KindService, key, instance); err != nil {
		return nil, err
	}
	return r.events.list(KindService, key.Namespace, key.Name, instance), nil
}

// GetMLServiceEvents returns resource-level runtime events for the Service.
func (r *Runtime) GetMLServiceEvents(_ context.Context, key types.NamespacedName) (*eventsv1.EventList, error) {
	return r.events.list(KindService, key.Namespace, key.Name, ""), nil
}

func protoString(p corev1.Protocol) string {
	switch p {
	case corev1.ProtocolUDP:
		return "udp"
	default:
		return "tcp"
	}
}
