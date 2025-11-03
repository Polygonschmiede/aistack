package configdir

import (
	"os"
	"path/filepath"
)

const defaultConfigDir = "/etc/aistack"

// ConfigDir resolves the configuration directory respecting overrides
func ConfigDir() string {
	if env := os.Getenv("AISTACK_CONFIG_DIR"); env != "" {
		if abs, err := filepath.Abs(env); err == nil {
			return abs
		}
	}
	return defaultConfigDir
}
