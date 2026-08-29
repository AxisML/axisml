package docker

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation"

	configapi "github.com/axisml/axisml/axisml-system/apis/pkg/workloadconfig"
)

// configMapSet is the workload-local ConfigMap namespace. Standalone has no
// Kubernetes API object store: spec.configMaps is the sole source of truth and
// references from the pod templates resolve against this set.
type configMapSet map[string]map[string]string

func newConfigMapSet(specs []configapi.ConfigMap) (configMapSet, error) {
	out := make(configMapSet, len(specs))
	for i, spec := range specs {
		if errs := validation.IsDNS1123Subdomain(spec.Name); len(errs) > 0 {
			return nil, capabilityError("configMaps[%d].name %q is invalid: %s", i, spec.Name, strings.Join(errs, "; "))
		}
		if _, exists := out[spec.Name]; exists {
			return nil, capabilityError("configMaps[%d].name %q is duplicated", i, spec.Name)
		}
		for key := range spec.Data {
			if errs := validation.IsConfigMapKey(key); len(errs) > 0 {
				return nil, capabilityError("configMaps[%d].data key %q is invalid: %s", i, key, strings.Join(errs, "; "))
			}
		}
		out[spec.Name] = spec.Data
	}
	return out, nil
}

// resolveEnv implements the ConfigMap-backed Kubernetes environment subset.
// envFrom sources are applied in order, and explicit env entries override them.
func resolveEnv(configMaps configMapSet, envFrom []corev1.EnvFromSource, env []corev1.EnvVar) ([]string, error) {
	values := map[string]string{}
	for i, source := range envFrom {
		if source.ConfigMapRef == nil {
			return nil, capabilityError("envFrom[%d] is unsupported in standalone (only configMapRef is supported)", i)
		}
		data, found, err := referencedConfigMap(configMaps, source.ConfigMapRef.Name, source.ConfigMapRef.Optional)
		if err != nil {
			return nil, fmt.Errorf("envFrom[%d]: %w", i, err)
		}
		if !found {
			continue
		}
		for key, value := range data {
			name := source.Prefix + key
			// Kubernetes skips ConfigMap keys that cannot form an environment
			// variable (the data remains available through a volume projection).
			if len(validation.IsEnvVarName(name)) != 0 {
				continue
			}
			values[name] = value
		}
	}

	for i, item := range env {
		if item.ValueFrom == nil {
			values[item.Name] = item.Value
			continue
		}
		selector := item.ValueFrom.ConfigMapKeyRef
		if selector == nil {
			return nil, capabilityError("env[%d].valueFrom is unsupported in standalone (only configMapKeyRef is supported)", i)
		}
		data, found, err := referencedConfigMap(configMaps, selector.Name, selector.Optional)
		if err != nil {
			return nil, fmt.Errorf("env[%d].valueFrom.configMapKeyRef: %w", i, err)
		}
		if !found {
			continue
		}
		value, exists := data[selector.Key]
		if !exists {
			if selector.Optional != nil && *selector.Optional {
				continue
			}
			return nil, capabilityError("ConfigMap %q has no key %q", selector.Name, selector.Key)
		}
		values[item.Name] = value
	}

	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]string, 0, len(names))
	for _, name := range names {
		out = append(out, name+"="+values[name])
	}
	return out, nil
}

func referencedConfigMap(configMaps configMapSet, name string, optional *bool) (map[string]string, bool, error) {
	data, found := configMaps[name]
	if found {
		return data, true, nil
	}
	if optional != nil && *optional {
		return nil, false, nil
	}
	return nil, false, capabilityError("ConfigMap %q is not declared in spec.configMaps", name)
}

// ConfigMapFile is one file in a ConfigMap volume projection.
type ConfigMapFile struct {
	Data string
	Mode uint32
}

func configMapFiles(configMaps configMapSet, source *corev1.ConfigMapVolumeSource) (map[string]ConfigMapFile, error) {
	data, found, err := referencedConfigMap(configMaps, source.Name, source.Optional)
	if err != nil {
		return nil, err
	}
	if !found {
		return map[string]ConfigMapFile{}, nil
	}

	defaultMode := int32(0o644)
	if source.DefaultMode != nil {
		defaultMode = *source.DefaultMode
	}
	if err := validateFileMode(defaultMode); err != nil {
		return nil, fmt.Errorf("ConfigMap %q defaultMode: %w", source.Name, err)
	}

	out := map[string]ConfigMapFile{}
	if len(source.Items) == 0 {
		for key, value := range data {
			if err := validateProjectionPath(key); err != nil {
				return nil, fmt.Errorf("ConfigMap %q key %q cannot be projected: %w", source.Name, key, err)
			}
			out[key] = ConfigMapFile{Data: value, Mode: uint32(defaultMode)}
		}
		return out, nil
	}

	for _, item := range source.Items {
		value, exists := data[item.Key]
		if !exists {
			if source.Optional != nil && *source.Optional {
				continue
			}
			return nil, capabilityError("ConfigMap %q has no key %q", source.Name, item.Key)
		}
		if err := validateProjectionPath(item.Path); err != nil {
			return nil, fmt.Errorf("ConfigMap %q item path %q: %w", source.Name, item.Path, err)
		}
		if _, duplicate := out[item.Path]; duplicate {
			return nil, capabilityError("ConfigMap %q projects more than one key to path %q", source.Name, item.Path)
		}
		mode := defaultMode
		if item.Mode != nil {
			mode = *item.Mode
		}
		if err := validateFileMode(mode); err != nil {
			return nil, fmt.Errorf("ConfigMap %q item %q mode: %w", source.Name, item.Key, err)
		}
		out[item.Path] = ConfigMapFile{Data: value, Mode: uint32(mode)}
	}
	return out, nil
}

func validateProjectionPath(value string) error {
	clean := path.Clean(value)
	if value == "" || path.IsAbs(value) || clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || clean != value {
		return capabilityError("must be a clean relative path that stays inside the volume")
	}
	return nil
}

func validateFileMode(mode int32) error {
	if mode < 0 || mode > 0o777 {
		return capabilityError("%#o is outside 0000-0777", mode)
	}
	return nil
}

func projectionPath(kind, namespace, name, role, volume string) (string, error) {
	parts := []string{kind, namespace, name, role, volume}
	for _, part := range parts {
		if part == "" || part == "." || part == ".." || strings.ContainsAny(part, `/\\`) {
			return "", capabilityError("ConfigMap projection path segment %q is invalid", part)
		}
	}
	return path.Join(parts...), nil
}

// reconcileConfigMapProjections materializes every projected ConfigMap volume.
// A manifest lets the steady-state reconcile skip filesystem writes while also
// removing obsolete projections when a desired spec changes before immutability
// is established by the compute layer.
func (r *Runtime) reconcileConfigMapProjections(kind, namespace, name string, plans []ContainerPlan) error {
	expected := map[string]map[string]ConfigMapFile{}
	for i := range plans {
		for _, mount := range plans[i].Mounts {
			if mount.ConfigMapPath == "" {
				continue
			}
			if current, exists := expected[mount.ConfigMapPath]; exists && projectionHash(current) != projectionHash(mount.ConfigMapFiles) {
				return capabilityError("ConfigMap projection %q has conflicting role definitions", mount.ConfigMapPath)
			}
			expected[mount.ConfigMapPath] = mount.ConfigMapFiles
		}
	}
	root, err := r.configMapWorkloadDir(kind, namespace, name)
	if err != nil {
		return err
	}
	if len(expected) == 0 {
		if root == "" {
			return nil
		}
		return os.RemoveAll(root)
	}
	if root == "" {
		return capabilityError("ConfigMap volume projection is not configured in standalone")
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return fmt.Errorf("create ConfigMap projection root: %w", err)
	}

	manifestPath := filepath.Join(root, ".axisml-manifest.json")
	previous := readProjectionManifest(manifestPath)
	next := make(map[string]string, len(expected))
	for relative, files := range expected {
		hash := projectionHash(files)
		next[relative] = hash
		dir, err := r.configMapProjectionDir(kind, namespace, name, relative)
		if err != nil {
			return err
		}
		if previous[relative] == hash && projectionDirMatches(dir, files) {
			continue
		}
		if err := rewriteProjectionDir(dir, files); err != nil {
			return fmt.Errorf("write ConfigMap projection %q: %w", relative, err)
		}
	}
	for relative := range previous {
		if _, keep := expected[relative]; keep {
			continue
		}
		dir, err := r.configMapProjectionDir(kind, namespace, name, relative)
		if err != nil {
			return err
		}
		if err := os.RemoveAll(dir); err != nil {
			return fmt.Errorf("remove obsolete ConfigMap projection %q: %w", relative, err)
		}
	}
	return writeProjectionManifest(manifestPath, next)
}

func (r *Runtime) removeConfigMapProjections(kind, namespace, name string) error {
	root, err := r.configMapWorkloadDir(kind, namespace, name)
	if err != nil {
		return err
	}
	if root == "" {
		return nil
	}
	return os.RemoveAll(root)
}

func (r *Runtime) configMapWorkloadDir(kind, namespace, name string) (string, error) {
	if r.cfg.ConfigMapsDir == "" {
		return "", nil
	}
	for _, part := range []string{kind, namespace, name} {
		if part == "" || part == "." || part == ".." || strings.ContainsAny(part, `/\\`) {
			return "", fmt.Errorf("invalid ConfigMap workload path segment %q", part)
		}
	}
	return filepath.Join(r.cfg.ConfigMapsDir, kind, namespace, name), nil
}

func (r *Runtime) configMapProjectionDir(kind, namespace, name, relative string) (string, error) {
	clean := path.Clean(relative)
	if path.IsAbs(relative) || clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || clean != relative {
		return "", fmt.Errorf("invalid ConfigMap projection manifest path %q", relative)
	}
	prefix := path.Join(kind, namespace, name) + "/"
	if !strings.HasPrefix(clean, prefix) {
		return "", fmt.Errorf("ConfigMap projection manifest path %q is outside workload %s/%s/%s", relative, kind, namespace, name)
	}
	return filepath.Join(r.cfg.ConfigMapsDir, filepath.FromSlash(clean)), nil
}

func projectionHash(files map[string]ConfigMapFile) string {
	body, _ := json.Marshal(files)
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])[:16]
}

func readProjectionManifest(filename string) map[string]string {
	body, err := os.ReadFile(filename)
	if err != nil {
		return map[string]string{}
	}
	var manifest map[string]string
	if json.Unmarshal(body, &manifest) != nil {
		return map[string]string{}
	}
	return manifest
}

func writeProjectionManifest(filename string, manifest map[string]string) error {
	body, err := json.Marshal(manifest)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(filename), ".axisml-manifest-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if _, err = tmp.Write(body); err == nil {
		err = tmp.Close()
	} else {
		_ = tmp.Close()
	}
	if err != nil {
		return err
	}
	return os.Rename(tmpName, filename)
}

func rewriteProjectionDir(dir string, files map[string]ConfigMapFile) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	// Preserve the projection directory inode: running containers may already
	// have this Docker volume subdirectory mounted. Only its children change.
	for _, entry := range entries {
		if err := os.RemoveAll(filepath.Join(dir, entry.Name())); err != nil {
			return err
		}
	}
	for relative, file := range files {
		filename := filepath.Join(dir, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(filename, []byte(file.Data), os.FileMode(file.Mode)); err != nil {
			return err
		}
		if err := os.Chmod(filename, os.FileMode(file.Mode)); err != nil {
			return err
		}
	}
	return nil
}

func projectionDirMatches(dir string, files map[string]ConfigMapFile) bool {
	seen := 0
	err := filepath.WalkDir(dir, func(filename string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(dir, filename)
		if err != nil {
			return err
		}
		desired, exists := files[filepath.ToSlash(relative)]
		if !exists {
			return fmt.Errorf("unexpected projection file")
		}
		body, err := os.ReadFile(filename)
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if string(body) != desired.Data || uint32(info.Mode().Perm()) != desired.Mode {
			return fmt.Errorf("projection file drift")
		}
		seen++
		return nil
	})
	return err == nil && seen == len(files)
}
