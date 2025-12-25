package pluginengine

import (
	"context"
	"errors"
	"testing"

	"scriptweaver/internal/core"
	"scriptweaver/internal/dag"
)

type recordingPlugin struct {
	manifest PluginManifest
	calls    *[]string

	panicBeforeRun  bool
	panicBeforeNode bool

	errBeforeRun  error
	errAfterRun   error
	errBeforeNode error
	errAfterNode  error
}

func (p *recordingPlugin) Manifest() PluginManifest { return p.manifest }

func (p *recordingPlugin) BeforeRun(context.Context) error {
	*p.calls = append(*p.calls, p.manifest.PluginID+":BeforeRun")
	if p.panicBeforeRun {
		panic("boom")
	}
	return p.errBeforeRun
}

func (p *recordingPlugin) AfterRun(context.Context) error {
	*p.calls = append(*p.calls, p.manifest.PluginID+":AfterRun")
	return p.errAfterRun
}

func (p *recordingPlugin) BeforeNode(_ context.Context, taskID string) error {
	*p.calls = append(*p.calls, p.manifest.PluginID+":BeforeNode:"+taskID)
	if p.panicBeforeNode {
		panic("boom")
	}
	return p.errBeforeNode
}

func (p *recordingPlugin) AfterNode(_ context.Context, taskID string) error {
	*p.calls = append(*p.calls, p.manifest.PluginID+":AfterNode:"+taskID)
	return p.errAfterNode
}

type okRunner struct{}

func (okRunner) Probe(context.Context, core.Task) (*dag.NodeResult, bool, error) {
	return nil, false, nil
}

func (okRunner) Run(context.Context, core.Task) (*dag.NodeResult, error) {
	return &dag.NodeResult{ExitCode: 0}, nil
}

func TestHookEngine_DeterministicOrderByPluginID(t *testing.T) {
	t.Parallel()

	var calls []string
	pB := &recordingPlugin{
		manifest: PluginManifest{PluginID: "b", Version: "0.1.0", Hooks: []string{"BeforeRun"}},
		calls:    &calls,
	}
	pA := &recordingPlugin{
		manifest: PluginManifest{PluginID: "a", Version: "0.1.0", Hooks: []string{"BeforeRun"}},
		calls:    &calls,
	}

	eng, err := NewHookEngine([]RuntimePlugin{pB, pA}, nil)
	if err != nil {
		t.Fatalf("NewHookEngine error: %v", err)
	}
	eng.BeforeRun(context.Background())

	if len(calls) != 2 {
		t.Fatalf("calls = %#v, want 2", calls)
	}
	if calls[0] != "a:BeforeRun" || calls[1] != "b:BeforeRun" {
		t.Fatalf("calls = %#v, want [a:BeforeRun b:BeforeRun]", calls)
	}
}

func TestExecutor_RunSerial_InvokesHookPoints(t *testing.T) {
	t.Parallel()

	g, err := dag.NewTaskGraph(
		[]core.Task{{Name: "A", Run: "run-a"}, {Name: "B", Run: "run-b"}},
		nil,
	)
	if err != nil {
		t.Fatalf("NewTaskGraph error: %v", err)
	}
	exec, err := dag.NewExecutor(g, okRunner{})
	if err != nil {
		t.Fatalf("NewExecutor error: %v", err)
	}

	var calls []string
	p := &recordingPlugin{
		manifest: PluginManifest{PluginID: "p", Version: "0.1.0", Hooks: []string{"BeforeRun", "AfterRun", "BeforeNode", "AfterNode"}},
		calls:    &calls,
	}
	eng, err := NewHookEngine([]RuntimePlugin{p}, nil)
	if err != nil {
		t.Fatalf("NewHookEngine error: %v", err)
	}
	exec.Hooks = eng

	_, runErr := exec.RunSerial(context.Background())
	if runErr != nil {
		t.Fatalf("RunSerial error: %v", runErr)
	}

	// Expect at least these boundaries. Node order is deterministic for this graph.
	want := []string{
		"p:BeforeRun",
		"p:BeforeNode:A",
		"p:AfterNode:A",
		"p:BeforeNode:B",
		"p:AfterNode:B",
		"p:AfterRun",
	}
	if len(calls) != len(want) {
		t.Fatalf("calls = %#v, want %#v", calls, want)
	}
	for i := range want {
		if calls[i] != want[i] {
			t.Fatalf("calls[%d]=%q, want %q (calls=%#v)", i, calls[i], want[i], calls)
		}
	}
}

func TestExecutor_RunSerial_MultiplePluginsSameHookDeterministic(t *testing.T) {
	t.Parallel()

	g, err := dag.NewTaskGraph(
		[]core.Task{{Name: "A", Run: "run-a"}},
		nil,
	)
	if err != nil {
		t.Fatalf("NewTaskGraph error: %v", err)
	}
	exec, err := dag.NewExecutor(g, okRunner{})
	if err != nil {
		t.Fatalf("NewExecutor error: %v", err)
	}

	var calls []string
	pB := &recordingPlugin{
		manifest: PluginManifest{PluginID: "b", Version: "0.1.0", Hooks: []string{"BeforeNode"}},
		calls:    &calls,
	}
	pA := &recordingPlugin{
		manifest: PluginManifest{PluginID: "a", Version: "0.1.0", Hooks: []string{"BeforeNode"}},
		calls:    &calls,
	}
	eng, err := NewHookEngine([]RuntimePlugin{pB, pA}, nil)
	if err != nil {
		t.Fatalf("NewHookEngine error: %v", err)
	}
	exec.Hooks = eng

	_, runErr := exec.RunSerial(context.Background())
	if runErr != nil {
		t.Fatalf("RunSerial error: %v", runErr)
	}

	// For a single node A: BeforeNode hooks should be invoked in plugin_id order.
	// AfterNode is not declared in the manifests, so only BeforeNode calls are asserted.
	want := []string{"a:BeforeNode:A", "b:BeforeNode:A"}
	if len(calls) < len(want) {
		t.Fatalf("calls = %#v, want at least %#v", calls, want)
	}
	if calls[0] != want[0] || calls[1] != want[1] {
		t.Fatalf("calls prefix = %#v, want %#v", calls[:2], want)
	}
}

func TestExecutor_RunSerial_PluginPanicRecovered(t *testing.T) {
	t.Parallel()

	g, err := dag.NewTaskGraph([]core.Task{{Name: "A", Run: "run-a"}}, nil)
	if err != nil {
		t.Fatalf("NewTaskGraph error: %v", err)
	}
	exec, err := dag.NewExecutor(g, okRunner{})
	if err != nil {
		t.Fatalf("NewExecutor error: %v", err)
	}

	var calls []string
	p := &recordingPlugin{
		manifest:       PluginManifest{PluginID: "p", Version: "0.1.0", Hooks: []string{"BeforeNode"}},
		calls:          &calls,
		panicBeforeNode: true,
	}
	eng, err := NewHookEngine([]RuntimePlugin{p}, nil)
	if err != nil {
		t.Fatalf("NewHookEngine error: %v", err)
	}
	exec.Hooks = eng

	_, runErr := exec.RunSerial(context.Background())
	if runErr != nil {
		t.Fatalf("RunSerial error: %v", runErr)
	}
	if len(eng.Errors()) == 0 {
		t.Fatalf("expected plugin panic to be recorded as error")
	}
}

func TestExecutor_RunSerial_PluginErrorDoesNotCrashEngine(t *testing.T) {
	t.Parallel()

	g, err := dag.NewTaskGraph([]core.Task{{Name: "A", Run: "run-a"}}, nil)
	if err != nil {
		t.Fatalf("NewTaskGraph error: %v", err)
	}
	exec, err := dag.NewExecutor(g, okRunner{})
	if err != nil {
		t.Fatalf("NewExecutor error: %v", err)
	}

	var calls []string
	p := &recordingPlugin{
		manifest:    PluginManifest{PluginID: "p", Version: "0.1.0", Hooks: []string{"AfterRun"}},
		calls:       &calls,
		errAfterRun: errors.New("hook failed"),
	}
	eng, err := NewHookEngine([]RuntimePlugin{p}, nil)
	if err != nil {
		t.Fatalf("NewHookEngine error: %v", err)
	}
	exec.Hooks = eng

	_, runErr := exec.RunSerial(context.Background())
	if runErr != nil {
		t.Fatalf("RunSerial error: %v", runErr)
	}
	if got := eng.Errors(); len(got) != 1 {
		t.Fatalf("Errors() = %#v, want 1 error", got)
	}
}
