//go:build cgo && linux

package standalone

import (
	"fmt"

	"github.com/NVIDIA/go-nvml/pkg/nvml"
)

// detectAllGPUs enumerates every GPU visible to NVML and returns their indices.
// It requires the NVIDIA driver to be injected into this container (the NVIDIA
// container runtime with NVIDIA_DRIVER_CAPABILITIES including "utility") so NVML
// can load libnvidia-ml.so.1 at runtime.
func detectAllGPUs() ([]int, error) {
	if ret := nvml.Init(); ret != nvml.SUCCESS {
		return nil, fmt.Errorf("nvml init: %s", nvml.ErrorString(ret))
	}
	defer func() { _ = nvml.Shutdown() }()

	count, ret := nvml.DeviceGetCount()
	if ret != nvml.SUCCESS {
		return nil, fmt.Errorf("nvml device count: %s", nvml.ErrorString(ret))
	}
	out := make([]int, 0, count)
	for i := 0; i < count; i++ {
		out = append(out, i)
	}
	return out, nil
}
