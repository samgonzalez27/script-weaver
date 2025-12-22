package integration

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type fileSnapshot struct {
	Mode fs.FileMode
	Size int64
	Hash string
}

// snapshotOutsideWorkspace records a deterministic snapshot of all regular files
// under projectRoot, excluding the .scriptweaver directory.
func snapshotOutsideWorkspace(projectRoot string) (map[string]fileSnapshot, error) {
	rootAbs, err := filepath.Abs(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve project root: %w", err)
	}

	snap := map[string]fileSnapshot{}
	walkErr := filepath.WalkDir(rootAbs, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// Exclude .scriptweaver subtree.
		if d.IsDir() && d.Name() == ".scriptweaver" {
			return filepath.SkipDir
		}
		if d.IsDir() {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}

		rel, err := filepath.Rel(rootAbs, path)
		if err != nil {
			return err
		}

		h, err := hashFile(path)
		if err != nil {
			return err
		}
		snap[filepath.ToSlash(rel)] = fileSnapshot{Mode: info.Mode(), Size: info.Size(), Hash: h}
		return nil
	})
	if walkErr != nil {
		return nil, fmt.Errorf("snapshot: %w", walkErr)
	}
	return snap, nil
}

func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func diffSnapshots(before, after map[string]fileSnapshot) string {
	changed := make([]string, 0)

	for path, b := range before {
		a, ok := after[path]
		if !ok {
			changed = append(changed, "removed "+path)
			continue
		}
		if b.Mode != a.Mode || b.Size != a.Size || b.Hash != a.Hash {
			changed = append(changed, "modified "+path)
		}
	}
	for path := range after {
		if _, ok := before[path]; !ok {
			changed = append(changed, "added "+path)
		}
	}

	sort.Strings(changed)
	return strings.Join(changed, "; ")
}
