package services

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"aistack/internal/configdir"
)

// ImageReference represents a resolved container image policy
type ImageReference struct {
	PullRef string
	TagRef  string
}

// VersionLock keeps deterministic image references per service
type VersionLock struct {
	entries map[string]string
	path    string
}

// Resolve returns the pull and tag references for a service
func (l *VersionLock) Resolve(serviceName, defaultImage string) (ImageReference, error) {
	if l == nil {
		return ImageReference{PullRef: defaultImage, TagRef: defaultImage}, nil
	}

	ref, ok := l.entries[serviceName]
	if !ok {
		return ImageReference{PullRef: defaultImage, TagRef: defaultImage}, nil
	}

	ref = strings.TrimSpace(ref)
	if ref == "" {
		return ImageReference{}, fmt.Errorf("versions.lock entry for %s is empty (file: %s)", serviceName, l.path)
	}

	if strings.Contains(ref, "@") {
		return ImageReference{PullRef: ref, TagRef: defaultImage}, nil
	}

	return ImageReference{PullRef: ref, TagRef: defaultImage}, nil
}

// loadVersionLock loads the versions.lock file if present
func loadVersionLock() (*VersionLock, error) {
	path := locateVersionLock()
	if path == "" {
		return nil, nil
	}

	file, err := os.Open(filepath.Clean(path)) // #nosec G304 -- path is derived from controlled configuration locations
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to open versions.lock: %w", err)
	}
	defer func() {
		if cerr := file.Close(); cerr != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to close versions.lock: %v\n", cerr)
		}
	}()

	scanner := bufio.NewScanner(file)
	entries := make(map[string]string)
	lineNo := 0

	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid versions.lock entry on line %d (file: %s): expected 'service:image[@digest]'", lineNo, path)
		}

		service := strings.TrimSpace(parts[0])
		ref := strings.TrimSpace(parts[1])

		if service == "" || ref == "" {
			return nil, fmt.Errorf("invalid versions.lock entry on line %d (file: %s): empty service or reference", lineNo, path)
		}

		entries[service] = ref
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read versions.lock: %w", err)
	}

	return &VersionLock{entries: entries, path: path}, nil
}

func locateVersionLock() string {
	if envPath := strings.TrimSpace(os.Getenv("AISTACK_VERSIONS_LOCK")); envPath != "" {
		if abs, err := filepath.Abs(envPath); err == nil {
			if fileExists(abs) {
				return abs
			}
		}
	}

	configCandidate := filepath.Join(configdir.ConfigDir(), "versions.lock")
	if fileExists(configCandidate) {
		return configCandidate
	}

	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		candidates := []string{
			filepath.Join(exeDir, "versions.lock"),
			filepath.Join(exeDir, "..", "share", "aistack", "versions.lock"),
		}
		for _, candidate := range candidates {
			if abs, err := filepath.Abs(candidate); err == nil && fileExists(abs) {
				return abs
			}
		}
	}

	if cwd, err := os.Getwd(); err == nil {
		candidate := filepath.Join(cwd, "versions.lock")
		if fileExists(candidate) {
			return candidate
		}
	}

	return ""
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
