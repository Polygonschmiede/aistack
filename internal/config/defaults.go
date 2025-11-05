package config

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() Config {
	return Config{
		ContainerRuntime: "docker",
		Profile:          "standard-gpu",
		GPULock:          true,
		Idle: IdleConfig{
			CPUIdleThreshold:   10,
			GPUIdleThreshold:   5,
			WindowSeconds:      300,
			IdleTimeoutSeconds: 1800, // 30 minutes
		},
		PowerEstimation: PowerConfig{
			BaselineWatts: 50,
		},
		WoL: WoLConfig{
			Interface: "eth0",
			MAC:       "00:00:00:00:00:00",
			RelayURL:  "",
		},
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
