package augment

import (
	"os"
	"path/filepath"
)

const defaultRoot = ".augment/sessions"

func DiscoverRoots(rootPath string) ([]string, error) {
	if rootPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		rootPath = filepath.Join(home, defaultRoot)
	}
	if info, err := os.Stat(rootPath); err != nil || !info.IsDir() {
		return nil, nil
	}
	return []string{rootPath}, nil
}

func DiscoverSessions(rootPath string) ([]string, error) {
	if rootPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		rootPath = filepath.Join(home, defaultRoot)
	}

	var paths []string
	entries, err := os.ReadDir(rootPath)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if filepath.Ext(e.Name()) != ".json" {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.Size() < 10 {
			continue
		}
		paths = append(paths, filepath.Join(rootPath, e.Name()))
	}
	return paths, nil
}
