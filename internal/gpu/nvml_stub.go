//go:build !cuda

package gpu

// NVMLInterface is a placeholder interface for builds without CUDA support.
type NVMLInterface interface{}

// DeviceInterface is a placeholder for builds without CUDA support.
type DeviceInterface interface{}

// NewRealNVML returns a nil placeholder when CUDA support is disabled.
func NewRealNVML() NVMLInterface {
	return nil
}
