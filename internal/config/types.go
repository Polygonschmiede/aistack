package config

// Config represents the complete aistack configuration
type Config struct {
	ContainerRuntime string        `yaml:"container_runtime"`
	Profile          string        `yaml:"profile"`
	GPULock          bool          `yaml:"gpu_lock"`
	Logging          LoggingConfig `yaml:"logging"`
	Models           ModelsConfig  `yaml:"models"`
	Updates          UpdatesConfig `yaml:"updates"`
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
