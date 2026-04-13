package session

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// LoadFromPath parses a specific JSONL file.
func LoadFromPath(jsonlPath string) (*Session, error) {
	return Parse(jsonlPath)
}

// LoadLast finds the most recently modified JSONL in the Claude projects directory.
func LoadLast() (*Session, error) {
	entries, err := findAllJSONL()
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, os.ErrNotExist
	}
	return Parse(entries[0])
}

// findAllJSONL returns all session JSONL files sorted by modification time (newest first).
func findAllJSONL() ([]string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	projectsDir := filepath.Join(home, ".claude", "projects")

	var files []struct {
		path    string
		modTime int64
	}

	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		return nil, err
	}
	for _, proj := range entries {
		if !proj.IsDir() {
			continue
		}
		projPath := filepath.Join(projectsDir, proj.Name())
		subs, err := os.ReadDir(projPath)
		if err != nil {
			continue
		}
		for _, sub := range subs {
			if sub.IsDir() || !strings.HasSuffix(sub.Name(), ".jsonl") {
				continue
			}
			info, err := sub.Info()
			if err != nil {
				continue
			}
			files = append(files, struct {
				path    string
				modTime int64
			}{
				path:    filepath.Join(projPath, sub.Name()),
				modTime: info.ModTime().UnixNano(),
			})
		}
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].modTime > files[j].modTime
	})

	paths := make([]string, len(files))
	for i, f := range files {
		paths[i] = f.path
	}
	return paths, nil
}
