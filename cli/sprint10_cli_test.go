package cli_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	icl "scriptweaver/internal/cli"
	"scriptweaver/internal/core"
	"scriptweaver/internal/dag"
	"scriptweaver/internal/recovery/state"
)

func writeGraphJSON(t *testing.T, path string, tasks []core.Task, edges []dag.Edge) {
	t.Helper()
	b, err := json.Marshal(map[string]any{"tasks": tasks, "edges": edges})
	if err != nil {
		t.Fatalf("marshal graph: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir graph dir: %v", err)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatalf("write graph: %v", err)
	}
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return b
}

func TestCLIValidation_NoSubcommandFails(t *testing.T) {
	res, err := icl.Run(context.Background(), nil)
	if err == nil {
		t.Fatalf("expected error")
	}
	if res.ExitCode != icl.ExitValidationError {
		t.Fatalf("expected exit %d got %d", icl.ExitValidationError, res.ExitCode)
	}
}

func TestCLIValidation_UnknownSubcommandFails(t *testing.T) {
	res, err := icl.Run(context.Background(), []string{"nope"})
	if err == nil {
		t.Fatalf("expected error")
	}
	if res.ExitCode != icl.ExitValidationError {
		t.Fatalf("expected exit %d got %d", icl.ExitValidationError, res.ExitCode)
	}
}

func TestCLIValidation_MissingRequiredFlagsStableMessage(t *testing.T) {
	res, err := icl.Run(context.Background(), []string{
		"run",
		"--workdir", "/tmp",
		"--cache-dir", "cache",
		"--output-dir", "out",
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	if res.ExitCode != icl.ExitValidationError {
		t.Fatalf("expected exit %d got %d", icl.ExitValidationError, res.ExitCode)
	}
	if !strings.Contains(err.Error(), "--graph is required") {
		t.Fatalf("expected missing --graph message, got %q", err.Error())
	}
}

func TestValidate_ValidGraphExits0_NoWorkspaceArtifacts(t *testing.T) {
	tmp := t.TempDir()
	graphPath := filepath.Join(tmp, "graph.json")
	writeGraphJSON(t, graphPath, []core.Task{{Name: "t1", Run: "true"}}, nil)

	res, err := icl.Run(context.Background(), []string{"validate", "--graph", graphPath})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.ExitCode != icl.ExitSuccess {
		t.Fatalf("expected exit 0 got %d", res.ExitCode)
	}
	if _, statErr := os.Stat(filepath.Join(tmp, ".scriptweaver")); !os.IsNotExist(statErr) {
		t.Fatalf("expected no workspace side-effects")
	}
}

func TestValidate_InvalidGraphExits1(t *testing.T) {
	tmp := t.TempDir()
	graphPath := filepath.Join(tmp, "graph.json")
	_ = os.WriteFile(graphPath, []byte(`{"tasks":[],"edges":[]}`), 0o644)

	res, err := icl.Run(context.Background(), []string{"validate", "--graph", graphPath})
	if err == nil {
		t.Fatalf("expected error")
	}
	if res.ExitCode != icl.ExitValidationError {
		t.Fatalf("expected exit %d got %d", icl.ExitValidationError, res.ExitCode)
	}
}

func TestRun_CleanCreatesWorkspace(t *testing.T) {
	workDir := t.TempDir()
	graphPath := filepath.Join(workDir, "graph.json")
	writeGraphJSON(t, graphPath, []core.Task{{
		Name:    "t1",
		Run:     "mkdir -p out && echo ok > out/x.txt",
		Outputs: []string{"out/x.txt"},
	}}, nil)

	res, err := icl.Run(context.Background(), []string{
		"run",
		"--workdir", workDir,
		"--graph", "graph.json",
		"--cache-dir", "cache",
		"--output-dir", "out",
		"--mode", "clean",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.ExitCode != icl.ExitSuccess {
		t.Fatalf("expected exit 0 got %d", res.ExitCode)
	}
	if _, err := os.Stat(filepath.Join(workDir, ".scriptweaver")); err != nil {
		t.Fatalf("expected workspace created: %v", err)
	}
}

func TestRun_IncrementalReuseCache(t *testing.T) {
	workDir := t.TempDir()
	graphPath := filepath.Join(workDir, "graph.json")
	writeGraphJSON(t, graphPath, []core.Task{{
		Name:    "t1",
		Run:     "if [ -f counter.txt ]; then echo X >> counter.txt; else echo X > counter.txt; fi; mkdir -p out && echo artifact > out/out.txt",
		Outputs: []string{"out/out.txt"},
	}}, nil)

	args := []string{
		"run",
		"--workdir", workDir,
		"--graph", "graph.json",
		"--cache-dir", "cache",
		"--output-dir", "out",
		"--mode", "incremental",
	}

	res1, err := icl.Run(context.Background(), args)
	if err != nil {
		t.Fatalf("run1 err: %v", err)
	}
	if res1.ExitCode != icl.ExitSuccess {
		t.Fatalf("run1 exit: %d", res1.ExitCode)
	}
	c1 := strings.TrimSpace(string(mustReadFile(t, filepath.Join(workDir, "counter.txt"))))
	if c1 != "X" {
		t.Fatalf("expected counter created once, got %q", c1)
	}

	res2, err := icl.Run(context.Background(), args)
	if err != nil {
		t.Fatalf("run2 err: %v", err)
	}
	if res2.ExitCode != icl.ExitSuccess {
		t.Fatalf("run2 exit: %d", res2.ExitCode)
	}
	c2 := strings.TrimSpace(string(mustReadFile(t, filepath.Join(workDir, "counter.txt"))))
	if c2 != "X" {
		t.Fatalf("expected task not to re-execute, got %q", c2)
	}
}

func TestResume_RequiresPriorRunAndLinksNewRun(t *testing.T) {
	workDir := t.TempDir()
	graphPath := filepath.Join(workDir, "graph.json")
	writeGraphJSON(t, graphPath, []core.Task{{Name: "t1", Run: "exit 7"}}, nil)

	res1, _ := icl.Run(context.Background(), []string{
		"run",
		"--workdir", workDir,
		"--graph", "graph.json",
		"--cache-dir", ".scriptweaver/cache",
		"--output-dir", "out",
		"--mode", "incremental",
	})
	if res1.ExitCode != icl.ExitExecutionError {
		t.Fatalf("expected failing run exit %d got %d", icl.ExitExecutionError, res1.ExitCode)
	}

	st, _ := state.NewStore(workDir)
	ids, err := st.ListRunIDs()
	if err != nil || len(ids) == 0 {
		t.Fatalf("expected at least one run id")
	}
	prevID := ids[len(ids)-1]

	res2, _ := icl.Run(context.Background(), []string{
		"resume",
		"--workdir", workDir,
		"--graph", "graph.json",
		"--previous-run-id", prevID,
	})
	if res2.ExitCode != icl.ExitExecutionError {
		t.Fatalf("expected resume to fail execution exit %d got %d", icl.ExitExecutionError, res2.ExitCode)
	}

	ids2, err := st.ListRunIDs()
	if err != nil || len(ids2) != len(ids)+1 {
		t.Fatalf("expected new run created")
	}
	linked := false
	for _, id := range ids2 {
		r, err := st.LoadRun(id)
		if err != nil {
			continue
		}
		if r.PreviousRunID != nil && *r.PreviousRunID == prevID {
			linked = true
			break
		}
	}
	if !linked {
		t.Fatalf("expected a run linked via previous_run_id")
	}
}

func TestResume_FailsIfGraphHashDiffers(t *testing.T) {
	workDir := t.TempDir()
	graphPath := filepath.Join(workDir, "graph.json")
	writeGraphJSON(t, graphPath, []core.Task{{Name: "t1", Run: "exit 7"}}, nil)

	_, _ = icl.Run(context.Background(), []string{
		"run",
		"--workdir", workDir,
		"--graph", "graph.json",
		"--cache-dir", ".scriptweaver/cache",
		"--output-dir", "out",
		"--mode", "incremental",
	})

	st, _ := state.NewStore(workDir)
	ids, _ := st.ListRunIDs()
	prevID := ids[len(ids)-1]

	graphPath2 := filepath.Join(workDir, "graph2.json")
	writeGraphJSON(t, graphPath2, []core.Task{{Name: "t2", Run: "true"}}, nil)

	res, err := icl.Run(context.Background(), []string{
		"resume",
		"--workdir", workDir,
		"--graph", "graph2.json",
		"--previous-run-id", prevID,
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	if res.ExitCode != icl.ExitValidationError {
		t.Fatalf("expected exit %d got %d", icl.ExitValidationError, res.ExitCode)
	}
}

func TestResume_WithoutPriorRunFails(t *testing.T) {
	workDir := t.TempDir()
	graphPath := filepath.Join(workDir, "graph.json")
	writeGraphJSON(t, graphPath, []core.Task{{Name: "t1", Run: "true"}}, nil)

	res, err := icl.Run(context.Background(), []string{
		"resume",
		"--workdir", workDir,
		"--graph", "graph.json",
		"--previous-run-id", "does-not-exist",
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	if res.ExitCode != icl.ExitValidationError {
		t.Fatalf("expected exit %d got %d", icl.ExitValidationError, res.ExitCode)
	}
}

func TestResume_DefaultRetriesAll_RetryFailedOnlyReusesCache(t *testing.T) {
	workDir := t.TempDir()
	graphPath := filepath.Join(workDir, "graph.json")

	writeGraphJSON(t, graphPath,
		[]core.Task{
			{
				Name:    "A",
				Run:     "if [ -f counter.txt ]; then echo X >> counter.txt; else echo X > counter.txt; fi; mkdir -p out && echo artifact > out/a.txt",
				Outputs: []string{"out/a.txt"},
			},
			{
				Name:   "B",
				Inputs: []string{"out/a.txt"},
				Run:    "exit 7",
			},
		},
		[]dag.Edge{{From: "A", To: "B"}},
	)

	_, _ = icl.Run(context.Background(), []string{
		"run",
		"--workdir", workDir,
		"--graph", "graph.json",
		"--cache-dir", ".scriptweaver/cache",
		"--output-dir", "out",
		"--mode", "incremental",
	})

	st, _ := state.NewStore(workDir)
	ids, _ := st.ListRunIDs()
	prevID := ids[len(ids)-1]

	// Default resume (retry-failed-only=false) re-executes A.
	_, _ = icl.Run(context.Background(), []string{
		"resume",
		"--workdir", workDir,
		"--graph", "graph.json",
		"--previous-run-id", prevID,
	})

	counterPath := filepath.Join(workDir, "counter.txt")
	counterRaw := string(mustReadFile(t, counterPath))
	counterLines := strings.Split(strings.TrimSpace(counterRaw), "\n")
	if len(counterLines) != 2 {
		t.Fatalf("expected A to re-run on default resume; got %q", strings.TrimSpace(counterRaw))
	}

	ids2, _ := st.ListRunIDs()
	prevID2 := ids2[len(ids2)-1]
	_, _ = icl.Run(context.Background(), []string{
		"resume",
		"--workdir", workDir,
		"--graph", "graph.json",
		"--previous-run-id", prevID2,
		"--retry-failed-only",
	})
	counterRaw2 := string(mustReadFile(t, counterPath))
	counterLines2 := strings.Split(strings.TrimSpace(counterRaw2), "\n")
	if len(counterLines2) != 2 {
		t.Fatalf("expected A not to re-run on retry-failed-only resume; got %q", strings.TrimSpace(counterRaw2))
	}
}
