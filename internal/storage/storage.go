package storage

import (
	"fmt"
	"io/fs"
	"path/filepath"
)

func ScanFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			files = append(files, path)
		}
		return nil
	})
    if err != nil {
        return nil, fmt.Errorf("failed to scan files: %w", err)
    }
    return files, nil
}
