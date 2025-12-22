package discovery

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"scriptweaver/internal/graph"
)

var (
	ErrNoGraphFound     = errors.New("no graph found")
	ErrAmbiguousGraphs  = errors.New("ambiguous graph discovery")
	ErrInvalidGraph     = errors.New("invalid graph")
	ErrInvalidGraphPath = errors.New("invalid graph path")
)

// Discover resolves a graph file path using a strict, deterministic precedence chain:
//  1) explicit CLI path (if provided)
//  2) <projectRoot>/graphs/
//  3) <projectRoot>/.scriptweaver/graphs/
//
// First match wins. If multiple candidates exist at the same precedence
// level, discovery fails.
//
// The returned path is absolute.
func Discover(projectRoot, explicitCLIPath string) (string, error) {
	root := strings.TrimSpace(projectRoot)
	if root == "" {
		return "", fmt.Errorf("%w: project root is required", ErrInvalidGraphPath)
	}

	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve project root: %w", err)
	}

	// 1) Explicit path
	if strings.TrimSpace(explicitCLIPath) != "" {
		p, err := resolveUnderRoot(rootAbs, explicitCLIPath)
		if err != nil {
			return "", err
		}
		if err := validateGraphFile(p); err != nil {
			return "", err
		}
		return p, nil
	}

	// 2) graphs/ at project root
	if p, ok, err := discoverSingleCandidate(filepath.Join(rootAbs, "graphs")); err != nil {
		return "", err
	} else if ok {
		if err := validateGraphFile(p); err != nil {
			return "", err
		}
		return p, nil
	}

	// 3) .scriptweaver/graphs/
	if p, ok, err := discoverSingleCandidate(filepath.Join(rootAbs, ".scriptweaver", "graphs")); err != nil {
		return "", err
	} else if ok {
		if err := validateGraphFile(p); err != nil {
			return "", err
		}
		return p, nil
	}

	return "", ErrNoGraphFound
}

func resolveUnderRoot(rootAbs, provided string) (string, error) {
	p := strings.TrimSpace(provided)
	if p == "" {
		return "", fmt.Errorf("%w: empty graph path", ErrInvalidGraphPath)
	}

	var abs string
	if filepath.IsAbs(p) {
		abs = filepath.Clean(p)
	} else {
		abs = filepath.Join(rootAbs, filepath.Clean(p))
	}

	abs, err := filepath.Abs(abs)
	if err != nil {
		return "", fmt.Errorf("%w: resolve path: %v", ErrInvalidGraphPath, err)
	}

	rel, err := filepath.Rel(rootAbs, abs)
	if err != nil {
		return "", fmt.Errorf("%w: resolve relative: %v", ErrInvalidGraphPath, err)
	}
	if rel == "." || (rel != "" && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != "..") {
		// ok
	} else {
		return "", fmt.Errorf("%w: path escapes project root", ErrInvalidGraphPath)
	}

	info, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidGraphPath, err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("%w: path is a directory", ErrInvalidGraphPath)
	}

	return abs, nil
}

func discoverSingleCandidate(dir string) (string, bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("read dir %s: %w", dir, err)
	}

	// Determinism: sort names before filtering.
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)

	candidates := make([]string, 0)
	for _, name := range names {
		full := filepath.Join(dir, name)
		info, err := os.Stat(full)
		if err != nil {
			return "", false, fmt.Errorf("stat candidate %s: %w", full, err)
		}
		if info.IsDir() {
			continue
		}
		candidates = append(candidates, full)
	}

	if len(candidates) == 0 {
		return "", false, nil
	}
	if len(candidates) > 1 {
		// Candidates are already in sorted order.
		return "", false, fmt.Errorf("%w: %s", ErrAmbiguousGraphs, strings.Join(candidates, ", "))
	}
	return candidates[0], true, nil
}

func validateGraphFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("%w: open %s: %v", ErrInvalidGraph, path, err)
	}
	defer func() { _ = f.Close() }()

	// graph.Parse enforces Sprint-06 schema (schema_version and unknown fields).
	if _, err := graph.Parse(io.Reader(f)); err != nil {
		return fmt.Errorf("%w: %s: %v", ErrInvalidGraph, path, err)
	}
	return nil
}
