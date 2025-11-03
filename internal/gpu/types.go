package gpu

// GPUInfo represents information about a single GPU
// Story T-009: GPU-Erkennung & NVML-Probe
type GPUInfo struct {
	Name     string `json:"name"`
	UUID     string `json:"uuid"`
	MemoryMB uint64 `json:"memory_mb"`
	Index    int    `json:"index"`
}

// GPUReport represents the complete GPU detection report
// Data Contract from EP-004: gpu_report.json
type GPUReport struct {
	DriverVersion string    `json:"driver_version"`
	CUDAVersion   int       `json:"cuda_version"`
	NVMLOk        bool      `json:"nvml_ok"`
	GPUs          []GPUInfo `json:"gpus"`
	ErrorMessage  string    `json:"error_message,omitempty"`
}

// ContainerToolkitReport represents NVIDIA Container Toolkit detection
// Story T-010: NVIDIA Container Toolkit Detection
type ContainerToolkitReport struct {
	DockerSupport  bool   `json:"docker_support"`
	ToolkitVersion string `json:"toolkit_version,omitempty"`
	ErrorMessage   string `json:"error_message,omitempty"`
}
