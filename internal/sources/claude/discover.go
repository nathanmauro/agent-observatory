package claude

import (
	"os"
	"path/filepath"
	"strings"
)

const defaultRoot = ".claude/projects"

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

// ProjectPathFromSource extracts the original filesystem path from the JSONL
// file's parent directory name. Claude Code encodes project paths by replacing
// each "/" with "-" in the absolute path, so "-Users-nathan-Developer-proj-name"
// represents "/Users/nathan/Developer/proj/name".
//
// The function walks up from the JSONL file to find the project-level directory
// (first directory whose name starts with "-") and decodes it by testing real
// filesystem paths to resolve the ambiguity between path separators and literal
// hyphens in directory names.
func ProjectPathFromSource(sourcePath string) string {
	dir := filepath.Dir(sourcePath)

	// Walk up past any intermediate dirs (subagents/, session UUID dirs)
	// to the project-level dir that starts with "-".
	for {
		base := filepath.Base(dir)
		if strings.HasPrefix(base, "-") {
			return decodeDashedPath(base)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

// decodeDashedPath resolves a dash-encoded path like "-Users-nathan-Developer-proj-name"
// by greedily testing the filesystem: at each dash position, prefer the interpretation
// where the segment is a real directory. Falls back to treating dashes as separators.
func decodeDashedPath(encoded string) string {
	if !strings.HasPrefix(encoded, "-") {
		return encoded
	}
	encoded = encoded[1:] // strip leading dash (represents leading /)

	var parts []string
	remaining := encoded

	for remaining != "" {
		dashIdx := strings.Index(remaining, "-")
		if dashIdx == -1 {
			parts = append(parts, remaining)
			break
		}

		// Greedily try to consume as much as possible before treating a dash as separator.
		// Check if taking more characters (past the next dash) yields a real directory.
		seg := remaining[:dashIdx]
		rest := remaining[dashIdx+1:]

		// Look ahead: is there a longer segment with a hyphen that matches a real dir?
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
