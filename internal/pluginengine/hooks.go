package pluginengine

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"scriptweaver/internal/dag"
)

// RuntimePlugin is the minimal runtime interface used for Phase 3 hook execution.
//
// The manifest declares which hooks are enabled. Implementations may support any
// subset of hook methods.
type RuntimePlugin interface {
	Manifest() PluginManifest
}

type beforeRunPlugin interface {
	BeforeRun(ctx context.Context) error
}

type afterRunPlugin interface {
	AfterRun(ctx context.Context) error
}

type beforeNodePlugin interface {
	BeforeNode(ctx context.Context, taskID string) error
}

type afterNodePlugin interface {
	AfterNode(ctx context.Context, taskID string) error
}

type pluginEntry struct {
	plugin RuntimePlugin
	id     string
	hooks  map[string]struct{}
}

// HookEngine executes registered plugin lifecycle hooks.
//
// Safety & isolation:
//   - recovers plugin panics
//   - logs and records plugin errors
//   - never returns hook errors to the core executor
//
// Determinism:
//   - plugins execute in stable order by plugin_id for each hook
type HookEngine struct {
	log Logger

	mu   sync.Mutex
	err  []error
	plug []pluginEntry
}

// NewHookEngine creates a HookEngine from runtime plugin implementations.
// Plugins are sorted by manifest plugin_id.
func NewHookEngine(plugins []RuntimePlugin, log Logger) (*HookEngine, error) {
	log = loggerOrNop(log)

	entries := make([]pluginEntry, 0, len(plugins))
	for _, p := range plugins {
		if p == nil {
			continue
		}
		m := p.Manifest()
		if err := ValidatePluginManifest(m); err != nil {
			return nil, err
		}
		hset := make(map[string]struct{}, len(m.Hooks))
		for _, h := range m.Hooks {
			hset[h] = struct{}{}
		}
		entries = append(entries, pluginEntry{plugin: p, id: m.PluginID, hooks: hset})
	}

	// Reject duplicate plugin IDs at runtime too.
	sort.Slice(entries, func(i, j int) bool { return entries[i].id < entries[j].id })
	for i := 1; i < len(entries); i++ {
		if entries[i].id == entries[i-1].id {
			return nil, fmt.Errorf("%w: %s", ErrDuplicatePluginID, entries[i].id)
		}
	}

	return &HookEngine{log: log, plug: entries}, nil
}

// Errors returns a snapshot of hook errors observed so far.
func (e *HookEngine) Errors() []error {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := make([]error, len(e.err))
	copy(out, e.err)
	return out
}

func (e *HookEngine) recordError(err error) {
	if err == nil {
		return
	}
	e.mu.Lock()
	e.err = append(e.err, err)
	e.mu.Unlock()
}

// --- dag.LifecycleHooks implementation ---
var _ dag.LifecycleHooks = (*HookEngine)(nil)

func (e *HookEngine) BeforeRun(ctx context.Context) {
	if e == nil {
		return
	}
	for _, ent := range e.plug {
		if _, ok := ent.hooks["BeforeRun"]; !ok {
			continue
		}
		p := ent.plugin
		h, ok := p.(beforeRunPlugin)
		if !ok {
			err := fmt.Errorf("plugin %s declares BeforeRun but does not implement it", ent.id)
			e.log.Printf("pluginengine: %v", err)
			e.recordError(err)
			continue
		}
		func() {
			defer func() {
				if r := recover(); r != nil {
					err := fmt.Errorf("plugin %s hook BeforeRun panic: %v", ent.id, r)
					e.log.Printf("pluginengine: %v", err)
					e.recordError(err)
				}
			}()
			if err := h.BeforeRun(ctx); err != nil {
				err2 := fmt.Errorf("plugin %s hook BeforeRun error: %w", ent.id, err)
				e.log.Printf("pluginengine: %v", err2)
				e.recordError(err2)
			}
		}()
	}
}

func (e *HookEngine) AfterRun(ctx context.Context) {
	if e == nil {
		return
	}
	for _, ent := range e.plug {
		if _, ok := ent.hooks["AfterRun"]; !ok {
			continue
		}
		p := ent.plugin
		h, ok := p.(afterRunPlugin)
		if !ok {
			err := fmt.Errorf("plugin %s declares AfterRun but does not implement it", ent.id)
			e.log.Printf("pluginengine: %v", err)
			e.recordError(err)
			continue
		}
		func() {
			defer func() {
				if r := recover(); r != nil {
					err := fmt.Errorf("plugin %s hook AfterRun panic: %v", ent.id, r)
					e.log.Printf("pluginengine: %v", err)
					e.recordError(err)
				}
			}()
			if err := h.AfterRun(ctx); err != nil {
				err2 := fmt.Errorf("plugin %s hook AfterRun error: %w", ent.id, err)
				e.log.Printf("pluginengine: %v", err2)
				e.recordError(err2)
			}
		}()
	}
}

func (e *HookEngine) BeforeNode(ctx context.Context, taskID string) {
	if e == nil {
		return
	}
	for _, ent := range e.plug {
		if _, ok := ent.hooks["BeforeNode"]; !ok {
			continue
		}
		p := ent.plugin
		h, ok := p.(beforeNodePlugin)
		if !ok {
			err := fmt.Errorf("plugin %s declares BeforeNode but does not implement it", ent.id)
			e.log.Printf("pluginengine: %v", err)
			e.recordError(err)
			continue
		}
		func() {
			defer func() {
				if r := recover(); r != nil {
					err := fmt.Errorf("plugin %s hook BeforeNode panic: %v", ent.id, r)
					e.log.Printf("pluginengine: %v", err)
					e.recordError(err)
				}
			}()
			if err := h.BeforeNode(ctx, taskID); err != nil {
				err2 := fmt.Errorf("plugin %s hook BeforeNode error: %w", ent.id, err)
				e.log.Printf("pluginengine: %v", err2)
				e.recordError(err2)
			}
		}()
	}
}

func (e *HookEngine) AfterNode(ctx context.Context, taskID string) {
	if e == nil {
		return
	}
	for _, ent := range e.plug {
		if _, ok := ent.hooks["AfterNode"]; !ok {
			continue
		}
		p := ent.plugin
		h, ok := p.(afterNodePlugin)
		if !ok {
			err := fmt.Errorf("plugin %s declares AfterNode but does not implement it", ent.id)
			e.log.Printf("pluginengine: %v", err)
			e.recordError(err)
			continue
		}
		func() {
			defer func() {
				if r := recover(); r != nil {
					err := fmt.Errorf("plugin %s hook AfterNode panic: %v", ent.id, r)
					e.log.Printf("pluginengine: %v", err)
					e.recordError(err)
				}
			}()
			if err := h.AfterNode(ctx, taskID); err != nil {
				err2 := fmt.Errorf("plugin %s hook AfterNode error: %w", ent.id, err)
				e.log.Printf("pluginengine: %v", err2)
				e.recordError(err2)
			}
		}()
	}
}
