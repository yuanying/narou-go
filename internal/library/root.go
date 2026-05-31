package library

import (
	"os"
	"path/filepath"
)

const localSettingDirName = ".narou"

// ResolveRoot returns the narou.rb library root.
func ResolveRoot(explicitPath string) (string, error) {
	if explicitPath != "" {
		return filepath.Abs(explicitPath)
	}

	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if info, err := os.Stat(filepath.Join(dir, localSettingDirName)); err == nil && info.IsDir() {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return filepath.Abs(".")
		}
		dir = parent
	}
}
