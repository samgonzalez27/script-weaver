package workspace

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Workspace describes the reserved ScriptWeaver workspace at a project root.
//
// The workspace is always located at <projectRoot>/.scriptweaver and is used to
// isolate ScriptWeaver state from user project files.
type Workspace struct {
	ProjectRoot string
	Dir         string
	CacheDir    string
	RunsDir     string
	LogsDir     string
	ConfigPath  string
}

var (
	ErrInvalidProjectRoot     = errors.New("invalid project root")
	ErrInvalidWorkspace       = errors.New("invalid .scriptweaver workspace")
	ErrUnauthorizedWorkspace  = errors.New("unauthorized entry in .scriptweaver")
	ErrWorkspacePathCollision = errors.New("workspace path exists but is not a directory")
)

// DetectProjectRoot returns the current working directory.
//
// Per spec, ScriptWeaver is invoked from a project root and the project root is
// the working directory. No environment-derived lookups are permitted.
func DetectProjectRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("detect project root: %w", err)
	}
	if wd == "" {
		return "", fmt.Errorf("detect project root: %w", ErrInvalidProjectRoot)
	}
	return wd, nil
}

// EnsureWorkspace validates and initializes the .scriptweaver workspace at the
// given project root.
//
// If projectRoot is empty, the current working directory is used.
//
// Zero-config behavior: if the workspace directory or required subdirectories do
// not exist, they are created.
//
// Rejection behavior: if the workspace contains any unauthorized files or
// directories (other than optional config.json), initialization fails.
func EnsureWorkspace(projectRoot string) (Workspace, error) {
	root := projectRoot
	if root == "" {
		var err error
		root, err = DetectProjectRoot()
		if err != nil {
			return Workspace{}, err
		}
	}

	workspaceDir := filepath.Join(root, ".scriptweaver")
	cacheDir := filepath.Join(workspaceDir, "cache")
	runsDir := filepath.Join(workspaceDir, "runs")
	logsDir := filepath.Join(workspaceDir, "logs")
	configPath := filepath.Join(workspaceDir, "config.json")

	ws := Workspace{
		ProjectRoot: root,
		Dir:         workspaceDir,
		CacheDir:    cacheDir,
		RunsDir:     runsDir,
		LogsDir:     logsDir,
		ConfigPath:  configPath,
	}

	info, err := os.Stat(workspaceDir)
	if err != nil {
		if !os.IsNotExist(err) {
			return Workspace{}, fmt.Errorf("stat workspace dir: %w", err)
		}
		if err := os.Mkdir(workspaceDir, 0o755); err != nil {
			return Workspace{}, fmt.Errorf("create workspace dir: %w", err)
		}
	} else if !info.IsDir() {
		return Workspace{}, fmt.Errorf("%w: %s", ErrWorkspacePathCollision, workspaceDir)
	}

	if err := validateWorkspaceTopLevel(workspaceDir); err != nil {
		return Workspace{}, err
	}

	// Create required directories if missing (zero-config).
	if err := ensureDir(cacheDir); err != nil {
		return Workspace{}, err
	}
	if err := ensureDir(runsDir); err != nil {
		return Workspace{}, err
	}
	if err := ensureDir(logsDir); err != nil {
		return Workspace{}, err
	}

	return ws, nil
}

func ensureDir(path string) error {
	info, err := os.Stat(path)
	if err == nil {
		if !info.IsDir() {
			return fmt.Errorf("%w: %s exists but is not a directory", ErrInvalidWorkspace, path)
		}
		return nil
	}
	if !os.IsNotExist(err) {
		return fmt.Errorf("stat dir %s: %w", path, err)
	}
	if err := os.MkdirAll(path, 0o755); err != nil {
		return fmt.Errorf("create dir %s: %w", path, err)
	}
	return nil
}

func validateWorkspaceTopLevel(workspaceDir string) error {
	entries, err := os.ReadDir(workspaceDir)
	if err != nil {
		return fmt.Errorf("read workspace dir: %w", err)
	}

	for _, entry := range entries {
		name := entry.Name()
		switch name {
		case "cache", "runs", "logs", "graphs":
			if !entry.IsDir() {
				return fmt.Errorf("%w: %s must be a directory", ErrInvalidWorkspace, filepath.Join(workspaceDir, name))
			}
		case "config.json":
			if entry.IsDir() {
				return fmt.Errorf("%w: %s must be a file", ErrInvalidWorkspace, filepath.Join(workspaceDir, name))
			}
		default:
			return fmt.Errorf("%w: %s", ErrUnauthorizedWorkspace, filepath.Join(workspaceDir, name))
		}
	}
	return nil
}
