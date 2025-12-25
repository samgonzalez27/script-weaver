package pluginengine

import (
	"errors"
	"io/fs"
)

var (
	// ErrManifestNotFound is matched via errors.Is(err, fs.ErrNotExist) and os.IsNotExist(err).
	ErrManifestNotFound = fs.ErrNotExist
	ErrManifestMalformed  = errors.New("manifest malformed")
	ErrManifestInvalid    = errors.New("manifest invalid")
	ErrDuplicatePluginID  = errors.New("duplicate plugin_id")
	ErrUnsupportedHook    = errors.New("unsupported hook")
	ErrMissingPluginID    = errors.New("missing plugin_id")
	ErrMissingVersion     = errors.New("missing version")
	ErrMissingHooks       = errors.New("missing hooks")
	ErrEmptyHooks         = errors.New("empty hooks")
)
