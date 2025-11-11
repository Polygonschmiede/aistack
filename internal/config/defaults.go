package config

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() Config {
	return Config{
		ContainerRuntime: "docker",
		Profile:          "standard-gpu",
		GPULock:          true,
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
		},
		Models: ModelsConfig{
			KeepCacheOnUninstall: true,
		},
		Updates: UpdatesConfig{
			Mode: "rolling",
		},
	}
}
