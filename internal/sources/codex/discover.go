package codex

import (
	"os"
	"path/filepath"
	"strings"
)

const defaultRoot = ".codex/sessions"

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
	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".jsonl" {
			return nil
		}
		if info.Size() < 10 {
			return nil
		}
		paths = append(paths, path)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return paths, nil
}

// ProjectPathFromSource extracts the cwd from the session_meta record.
// Codex session paths encode date but not project: YYYY/MM/DD/rollout-*.jsonl.
// The actual project path comes from parsing session_meta.payload.cwd.
// We return empty here; the parser fills it in via session metadata.
func ProjectPathFromSource(sourcePath string) string {
	// Codex filenames are like rollout-2026-05-27T11-13-52-UUID.jsonl
	// No project path is encoded in the path structure.
	// The indexer will use the cwd from session_meta instead.
	base := filepath.Base(sourcePath)
	if strings.HasPrefix(base, "rollout-") {
		return ""
	}
	return ""
}
