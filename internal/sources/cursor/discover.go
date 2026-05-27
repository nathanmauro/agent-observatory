package cursor

import (
	"os"
	"path/filepath"
	"strings"
)


const defaultRoot = ".cursor/projects"

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

// ProjectPathFromSource extracts the project path from a Cursor transcript path.
// Cursor uses the same dash-encoding as Claude Code for project directories:
// ~/.cursor/projects/<encoded-path>/agent-transcripts/<uuid>/<uuid>.jsonl
func ProjectPathFromSource(sourcePath string) string {
	dir := filepath.Dir(sourcePath)

	for {
		base := filepath.Base(dir)
		parent := filepath.Dir(dir)

		if filepath.Base(parent) == "projects" && strings.Contains(filepath.Dir(parent), ".cursor") {
			return decodeDashedPath(base)
		}

		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

// decodeDashedPath resolves a dash-encoded path like "Users-nathan-Developer-proj-name"
// by greedily testing the filesystem to distinguish path separators from literal hyphens.
func decodeDashedPath(encoded string) string {
	if encoded == "" || encoded == "empty-window" {
		return encoded
	}

	// Cursor encodes without leading dash (unlike Claude which uses "-Users-...")
	working := encoded
	if strings.HasPrefix(working, "-") {
		working = working[1:]
	}

	var parts []string
	remaining := working

	for remaining != "" {
		dashIdx := strings.Index(remaining, "-")
		if dashIdx == -1 {
			parts = append(parts, remaining)
			break
		}

		seg := remaining[:dashIdx]
		rest := remaining[dashIdx+1:]

		bestSeg := seg
		bestRest := rest
		scan := rest
		for {
			nextDash := strings.Index(scan, "-")
			if nextDash == -1 {
				candidate := remaining
				testPath := "/" + strings.Join(append(parts, candidate), "/")
				if info, err := os.Stat(testPath); err == nil && info.IsDir() {
					bestSeg = candidate
					bestRest = ""
				}
				break
			}
			candidate := remaining[:dashIdx+1+nextDash]
			testPath := "/" + strings.Join(append(parts, candidate), "/")
			if info, err := os.Stat(testPath); err == nil && info.IsDir() {
				bestSeg = candidate
				bestRest = scan[nextDash+1:]
			}
			scan = scan[nextDash+1:]
			dashIdx = dashIdx + 1 + nextDash
		}

		parts = append(parts, bestSeg)
		remaining = bestRest
	}

	return "/" + strings.Join(parts, "/")
}
