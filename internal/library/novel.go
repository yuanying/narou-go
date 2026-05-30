package library

import "path/filepath"

// NovelDir returns the archive directory for a database entry.
func NovelDir(libraryPath string, entry NovelEntry) string {
	return filepath.Join(libraryPath, "小説データ", entry.SiteName, entry.FileTitle)
}
