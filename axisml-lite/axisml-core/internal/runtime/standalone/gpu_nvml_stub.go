//go:build !cgo || !linux

package standalone

import "fmt"

// detectAllGPUs is unavailable on non-linux or non-cgo builds: NVML autodetection
// needs a linux build with cgo. Configure explicit device indices instead.
func detectAllGPUs() ([]int, error) {
	return nil, fmt.Errorf("AXISML_GPU_DEVICES=all requires a linux build with NVML (cgo); set explicit device indices instead")
}
