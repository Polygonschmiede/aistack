package config

// Config represents the complete aistack configuration
type Config struct {
	ContainerRuntime string                `yaml:"container_runtime"`
	Profile          string                `yaml:"profile"`
	GPULock          bool                  `yaml:"gpu_lock"`
	Idle             IdleConfig            `yaml:"idle"`
	PowerEstimation  PowerEstimationConfig `yaml:"power_estimation"`
	WoL              WoLConfig             `yaml:"wol"`
	Logging          LoggingConfig         `yaml:"logging"`
	Models           ModelsConfig          `yaml:"models"`
	Updates          UpdatesConfig         `yaml:"updates"`
}

// IdleConfig represents idle detection configuration
type IdleConfig struct {
	CPUIdleThreshold   int `yaml:"cpu_idle_threshold"`
	GPUIdleThreshold   int `yaml:"gpu_idle_threshold"`
	WindowSeconds      int `yaml:"window_seconds"`
	IdleTimeoutSeconds int `yaml:"idle_timeout_seconds"`
}

// PowerEstimationConfig represents power estimation configuration
type PowerEstimationConfig struct {
	BaselineWatts float64 `yaml:"baseline_watts"`
}

// WoLConfig represents Wake-on-LAN configuration
type WoLConfig struct {
	Interface string `yaml:"interface"`
	MAC       string `yaml:"mac"`
}

// LoggingConfig represents logging configuration
type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

// ModelsConfig represents model cache management configuration
type ModelsConfig struct {
	KeepCacheOnUninstall bool `yaml:"keep_cache_on_uninstall"`
}

// UpdatesConfig represents update policy configuration
type UpdatesConfig struct {
	Mode string `yaml:"mode"`
}

// ValidationError represents a configuration validation error
type ValidationError struct {
	Path    string
	Message string
}

func (e ValidationError) Error() string {
	return e.Path + ": " + e.Message
}
