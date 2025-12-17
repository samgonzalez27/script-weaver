package incremental

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"sort"
)

// InvalidationReasonType is the stable reason category.
//
// IMPORTANT: These string values are part of the canonical serialized bytes.
// Do not rename.
//
// Source: docs/sprints/sprint-04/in-process/invalidation-engine/spec.md and
// docs/sprints/sprint-04/planning/data-dictionary.md.
type InvalidationReasonType string

const (
	ReasonTypeInputChanged          InvalidationReasonType = "InputChanged"
	ReasonTypeEnvChanged            InvalidationReasonType = "EnvChanged"
	ReasonTypeDependencyInvalidated InvalidationReasonType = "DependencyInvalidated"
	ReasonTypeGraphStructureChanged InvalidationReasonType = "GraphStructureChanged"
	ReasonTypeCommandChanged        InvalidationReasonType = "CommandChanged"
	ReasonTypeOutputChanged         InvalidationReasonType = "OutputChanged"
)

// InvalidationDetail is an optional key/value pair providing context specific to a reason.
//
// Data dictionary notes Details as an optional map/string. We store it as a sorted slice of
// key/value pairs to avoid non-deterministic map iteration.
type InvalidationDetail struct {
	Key   string
	Value string
}

// InvalidationReason describes a single atomic cause for task invalidation.
//
// Canonicalization invariants:
//   - Type is required.
//   - SourceTaskID is required IFF Type == DependencyInvalidated.
//   - Details are sorted by (Key, Value) and deduplicated.
//
// Determinism invariant: Canonical reasons must serialize to identical bytes regardless
// of creation order (including Details ordering).
type InvalidationReason struct {
	Type InvalidationReasonType

	// SourceTaskID is the root-cause task ID (required for DependencyInvalidated).
	SourceTaskID string

	// Details is optional, type-specific context (e.g., InputName, EnvName).
	Details []InvalidationDetail
}

func (r InvalidationReason) Validate() error {
	if r.Type == "" {
		return errors.New("invalidation reason type is required")
	}
	if r.Type == ReasonTypeDependencyInvalidated && r.SourceTaskID == "" {
		return errors.New("dependency invalidation requires sourceTaskID")
	}
	for i := range r.Details {
		if r.Details[i].Key == "" {
			return fmt.Errorf("details[%d].key is empty", i)
		}
	}
	return nil
}

func (r InvalidationReason) canonical() InvalidationReason {
	if len(r.Details) == 0 {
		r.Details = nil
		return r
	}
	dd := make([]InvalidationDetail, 0, len(r.Details))
	dd = append(dd, r.Details...)
	sort.Slice(dd, func(i, j int) bool {
		if dd[i].Key != dd[j].Key {
			return dd[i].Key < dd[j].Key
		}
		return dd[i].Value < dd[j].Value
	})
	// Deduplicate.
	j := 0
	for i := 0; i < len(dd); i++ {
		if i == 0 || dd[i] != dd[i-1] {
			dd[j] = dd[i]
			j++
		}
	}
	r.Details = dd[:j]
	return r
}

func (r InvalidationReason) MarshalBinary() ([]byte, error) {
	r = r.canonical()
	if err := r.Validate(); err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	// Fixed field order encoding:
	//   type:string
	//   hasSource:uint8
	//   sourceTaskId:string (only if hasSource)
	//   detailsCount:uint32
	//   details: (key:string, value:string)*
	writeString(&buf, string(r.Type))
	if r.SourceTaskID != "" {
		buf.WriteByte(1)
		writeString(&buf, r.SourceTaskID)
	} else {
		buf.WriteByte(0)
	}
	binary.Write(&buf, binary.BigEndian, uint32(len(r.Details)))
	for _, d := range r.Details {
		writeString(&buf, d.Key)
		writeString(&buf, d.Value)
	}
	return buf.Bytes(), nil
}

// InvalidationReasons is a per-task set of reasons.
//
// Canonicalization sorts and deduplicates reasons so ordering is stable and creation-order independent.
type InvalidationReasons []InvalidationReason

func (rs InvalidationReasons) Canonicalize() InvalidationReasons {
	if len(rs) == 0 {
		return nil
	}
	out := make([]InvalidationReason, 0, len(rs))
	for _, r := range rs {
		out = append(out, r.canonical())
	}
	sort.Slice(out, func(i, j int) bool {
		a := out[i]
		b := out[j]
		if reasonTypeOrder(a.Type) != reasonTypeOrder(b.Type) {
			return reasonTypeOrder(a.Type) < reasonTypeOrder(b.Type)
		}
		if a.SourceTaskID != b.SourceTaskID {
			return a.SourceTaskID < b.SourceTaskID
		}
		return compareDetails(a.Details, b.Details)
	})
	// Deduplicate identical canonical reasons.
	j := 0
	for i := 0; i < len(out); i++ {
		if i == 0 || !reasonEqual(out[i], out[i-1]) {
			out[j] = out[i]
			j++
		}
	}
	return out[:j]
}

func (rs InvalidationReasons) MarshalBinary() ([]byte, error) {
	rs = rs.Canonicalize()
	var buf bytes.Buffer
	binary.Write(&buf, binary.BigEndian, uint32(len(rs)))
	for _, r := range rs {
		b, err := r.MarshalBinary()
		if err != nil {
			return nil, err
		}
		binary.Write(&buf, binary.BigEndian, uint32(len(b)))
		buf.Write(b)
	}
	return buf.Bytes(), nil
}

func reasonTypeOrder(t InvalidationReasonType) int {
	switch t {
	case ReasonTypeInputChanged:
		return 10
	case ReasonTypeEnvChanged:
		return 20
	case ReasonTypeDependencyInvalidated:
		return 30
	case ReasonTypeGraphStructureChanged:
		return 40
	case ReasonTypeCommandChanged:
		return 50
	case ReasonTypeOutputChanged:
		return 60
	default:
		return 1000
	}
}

func compareDetails(a, b []InvalidationDetail) bool {
	la := len(a)
	lb := len(b)
	min := la
	if lb < min {
		min = lb
	}
	for i := 0; i < min; i++ {
		if a[i].Key != b[i].Key {
			return a[i].Key < b[i].Key
		}
		if a[i].Value != b[i].Value {
			return a[i].Value < b[i].Value
		}
	}
	return la < lb
}

func reasonEqual(a, b InvalidationReason) bool {
	if a.Type != b.Type || a.SourceTaskID != b.SourceTaskID {
		return false
	}
	if len(a.Details) != len(b.Details) {
		return false
	}
	for i := range a.Details {
		if a.Details[i] != b.Details[i] {
			return false
		}
	}
	return true
}

func writeString(buf *bytes.Buffer, s string) {
	binary.Write(buf, binary.BigEndian, uint32(len(s)))
	buf.WriteString(s)
}

// NodeSnapshot captures the minimal identity inputs required to decide whether a node
// is unchanged or invalidated.
//
// This intentionally keeps "input content" distinct from "declared inputs".
// The incremental engine can compute InputHash from resolved file contents, while
// DeclaredInputs records the task's declared input set.
type NodeSnapshot struct {
	Name string

	// TaskHash is the deterministic execution/cache identity for the node.
	// It is used by incremental planning to check cache presence.
	TaskHash string

	// DeclaredInputs is the task's declared input set (paths/globs).
	// It is treated as a set for identity.
	DeclaredInputs []string

	// InputHash is a deterministic summary of resolved input content.
	// Any change must invalidate the node.
	InputHash string

	// Env is the task's declared environment variable map.
	Env map[string]string

	// Command is the task's command string.
	Command string

	// Outputs is the task's declared outputs.
	// It is treated as a set for identity.
	Outputs []string

	// Upstream is the list of direct dependency node names.
	// It is treated as a set for identity.
	Upstream []string
}

// GraphSnapshot represents the minimal information needed to compute an incremental invalidation plan.
//
// Nodes are addressed by stable node name.
type GraphSnapshot struct {
	Nodes map[string]NodeSnapshot
}

// InvalidationEntry is the per-node invalidation decision.
type InvalidationEntry struct {
	Invalidated bool
	Reasons     InvalidationReasons
}

// InvalidationMap maps node name -> invalidation decision.
//
// It includes entries for every node in newGraph.
type InvalidationMap map[string]InvalidationEntry

// MarshalBinary returns a deterministic binary encoding of the invalidation map.
//
// Determinism strategy:
//   - Sort task IDs lexicographically.
//   - For each task ID, serialize (taskID, reasonsBytes) with length prefixes.
//   - Reasons are serialized using their own canonicalization and deterministic encoding.
//
// This avoids relying on Go map iteration order.
func (m InvalidationMap) MarshalBinary() ([]byte, error) {
	if len(m) == 0 {
		return nil, nil
	}

	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var buf bytes.Buffer
	binary.Write(&buf, binary.BigEndian, uint32(len(keys)))
	for _, k := range keys {
		e := m[k]
		writeString(&buf, k)
		reasonsBytes, err := e.Reasons.MarshalBinary()
		if err != nil {
			return nil, err
		}
		binary.Write(&buf, binary.BigEndian, uint32(len(reasonsBytes)))
		buf.Write(reasonsBytes)
	}
	return buf.Bytes(), nil
}

// CalculateInvalidation computes which nodes in newGraph are invalidated relative to oldGraph.
//
// Invalidation is strictly transitive: if A is invalidated, every downstream dependent of A
// in the new graph is invalidated as well.
func CalculateInvalidation(oldGraph, newGraph *GraphSnapshot) InvalidationMap {
	result := make(InvalidationMap)
	if newGraph == nil || len(newGraph.Nodes) == 0 {
		return result
	}

	oldNodes := map[string]NodeSnapshot{}
	if oldGraph != nil && oldGraph.Nodes != nil {
		oldNodes = oldGraph.Nodes
	}

	// Canonical node name list.
	names := make([]string, 0, len(newGraph.Nodes))
	for name := range newGraph.Nodes {
		names = append(names, name)
	}
	sort.Strings(names)

	// Build deterministic adjacency + indegrees from the new graph.
	outgoing := make(map[string][]string, len(newGraph.Nodes))
	indeg := make(map[string]int, len(newGraph.Nodes))
	for _, name := range names {
		indeg[name] = 0
	}
	for _, name := range names {
		n := newGraph.Nodes[name]
		for _, parent := range normalizeStringSet(n.Upstream) {
			// A missing upstream dependency is considered a graph-structure change for this node.
			if _, exists := newGraph.Nodes[parent]; !exists {
				continue
			}
			outgoing[parent] = append(outgoing[parent], name)
			indeg[name]++
		}
	}
	for k := range outgoing {
		sort.Strings(outgoing[k])
	}

	// Deterministic topological order (lexical tie-break).
	topo := topoOrder(names, outgoing, indeg)

	// Root-cause tracking for dependency propagation.
	rootSources := make(map[string][]string, len(newGraph.Nodes))

	directReasonsFor := func(taskID string, oldNode NodeSnapshot, existed bool, newNode NodeSnapshot) InvalidationReasons {
		var direct InvalidationReasons
		if !existed {
			return InvalidationReasons{InvalidationReason{Type: ReasonTypeGraphStructureChanged}}.Canonicalize()
		}

		if newNode.InputHash != oldNode.InputHash {
			direct = append(direct, InvalidationReason{Type: ReasonTypeInputChanged})
		}

		// Graph structure changes: declared inputs set changes are treated as graph structure changes for sprint-04.
		if !equalStringSet(newNode.DeclaredInputs, oldNode.DeclaredInputs) {
			for _, name := range symmetricSetDiff(oldNode.DeclaredInputs, newNode.DeclaredInputs) {
				direct = append(direct, InvalidationReason{Type: ReasonTypeGraphStructureChanged, Details: []InvalidationDetail{{Key: "InputName", Value: name}}})
			}
			if len(direct) == 0 {
				direct = append(direct, InvalidationReason{Type: ReasonTypeGraphStructureChanged, Details: []InvalidationDetail{{Key: "DeclaredInputs", Value: "changed"}}})
			}
		}

		if !equalStringMap(newNode.Env, oldNode.Env) {
			keys := changedMapKeys(oldNode.Env, newNode.Env)
			if len(keys) == 0 {
				direct = append(direct, InvalidationReason{Type: ReasonTypeEnvChanged})
			} else {
				details := make([]InvalidationDetail, 0, len(keys))
				for _, k := range keys {
					details = append(details, InvalidationDetail{Key: "EnvName", Value: k})
				}
				direct = append(direct, InvalidationReason{Type: ReasonTypeEnvChanged, Details: details})
			}
		}

		if newNode.Command != oldNode.Command {
			direct = append(direct, InvalidationReason{Type: ReasonTypeCommandChanged})
		}

		// OutputChanged includes declared output set changes. File-existence checks are outside the snapshot scope.
		if !equalStringSet(newNode.Outputs, oldNode.Outputs) {
			outputs := symmetricSetDiff(oldNode.Outputs, newNode.Outputs)
			if len(outputs) == 0 {
				direct = append(direct, InvalidationReason{Type: ReasonTypeOutputChanged})
			} else {
				details := make([]InvalidationDetail, 0, len(outputs))
				for _, o := range outputs {
					details = append(details, InvalidationDetail{Key: "OutputName", Value: o})
				}
				direct = append(direct, InvalidationReason{Type: ReasonTypeOutputChanged, Details: details})
			}
		}

		// Upstream dependency identity (direct parents) is compared as a set.
		if !equalStringSet(newNode.Upstream, oldNode.Upstream) {
			direct = append(direct, InvalidationReason{Type: ReasonTypeGraphStructureChanged, Details: []InvalidationDetail{{Key: "Upstream", Value: "changed"}}})
		}

		// Missing upstream dependency in the new graph is a structural change for this node.
		for _, parent := range normalizeStringSet(newNode.Upstream) {
			if _, ok := newGraph.Nodes[parent]; !ok {
				direct = append(direct, InvalidationReason{Type: ReasonTypeGraphStructureChanged, Details: []InvalidationDetail{{Key: "UpstreamTaskID", Value: parent}, {Key: "Upstream", Value: "missing"}}})
			}
		}

		_ = taskID // reserved for future detail expansion
		return direct.Canonicalize()
	}

	// Compute reasons in deterministic topological order.
	for _, name := range topo {
		newNode := newGraph.Nodes[name]
		oldNode, existed := oldNodes[name]

		direct := directReasonsFor(name, oldNode, existed, newNode)

		// Dependency invalidation reasons reference root causes.
		sourceSet := make(map[string]struct{})
		for _, parent := range normalizeStringSet(newNode.Upstream) {
			pEntry, ok := result[parent]
			if !ok || !pEntry.Invalidated {
				continue
			}
			for _, src := range rootSources[parent] {
				sourceSet[src] = struct{}{}
			}
		}

		depSources := make([]string, 0, len(sourceSet))
		for src := range sourceSet {
			depSources = append(depSources, src)
		}
		sort.Strings(depSources)

		var dep InvalidationReasons
		for _, src := range depSources {
			dep = append(dep, InvalidationReason{Type: ReasonTypeDependencyInvalidated, SourceTaskID: src})
		}

		reasons := append(direct, dep...).Canonicalize()
		entry := InvalidationEntry{Invalidated: len(reasons) > 0, Reasons: reasons}
		result[name] = entry

		// Compute this node's root causes for downstream propagation.
		if !entry.Invalidated {
			rootSources[name] = nil
			continue
		}

		rootSet := make(map[string]struct{})
		// If any direct reason exists (i.e., non-dependency root causes), include self.
		if len(direct) > 0 {
			rootSet[name] = struct{}{}
		}
		// If the node is invalidated due to upstream roots, propagate those roots.
		for _, src := range depSources {
			rootSet[src] = struct{}{}
		}
		rootList := make([]string, 0, len(rootSet))
		for src := range rootSet {
			rootList = append(rootList, src)
		}
		sort.Strings(rootList)
		rootSources[name] = rootList
	}

	return result
}

func symmetricSetDiff(a, b []string) []string {
	aa := normalizeStringSet(a)
	bb := normalizeStringSet(b)

	setA := make(map[string]struct{}, len(aa))
	for _, v := range aa {
		setA[v] = struct{}{}
	}
	setB := make(map[string]struct{}, len(bb))
	for _, v := range bb {
		setB[v] = struct{}{}
	}

	var diff []string
	for _, v := range aa {
		if _, ok := setB[v]; !ok {
			diff = append(diff, v)
		}
	}
	for _, v := range bb {
		if _, ok := setA[v]; !ok {
			diff = append(diff, v)
		}
	}
	sort.Strings(diff)
	return diff
}

func changedMapKeys(a, b map[string]string) []string {
	if len(a) == 0 && len(b) == 0 {
		return nil
	}
	keys := make(map[string]struct{}, len(a)+len(b))
	for k := range a {
		keys[k] = struct{}{}
	}
	for k := range b {
		keys[k] = struct{}{}
	}
	all := make([]string, 0, len(keys))
	for k := range keys {
		all = append(all, k)
	}
	sort.Strings(all)
	var changed []string
	for _, k := range all {
		av, aok := a[k]
		bv, bok := b[k]
		if aok != bok || av != bv {
			changed = append(changed, k)
		}
	}
	return changed
}

func normalizeStringSet(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, 0, len(in))
	out = append(out, in...)
	sort.Strings(out)
	// Deduplicate.
	j := 0
	for i := 0; i < len(out); i++ {
		if i == 0 || out[i] != out[i-1] {
			out[j] = out[i]
			j++
		}
	}
	return out[:j]
}

func equalStringSet(a, b []string) bool {
	aa := normalizeStringSet(a)
	bb := normalizeStringSet(b)
	if len(aa) != len(bb) {
		return false
	}
	for i := range aa {
		if aa[i] != bb[i] {
			return false
		}
	}
	return true
}

func equalStringMap(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, av := range a {
		bv, ok := b[k]
		if !ok {
			return false
		}
		if av != bv {
			return false
		}
	}
	return true
}

func topoOrder(names []string, outgoing map[string][]string, indeg map[string]int) []string {
	// Work on a copy.
	ind := make(map[string]int, len(indeg))
	for k, v := range indeg {
		ind[k] = v
	}

	ready := make([]string, 0, len(names))
	for _, n := range names {
		if ind[n] == 0 {
			ready = append(ready, n)
		}
	}
	sort.Strings(ready)

	order := make([]string, 0, len(names))
	for len(ready) > 0 {
		n := ready[0]
		ready = ready[1:]
		order = append(order, n)

		for _, m := range outgoing[n] {
			ind[m]--
			if ind[m] == 0 {
				// Insert m into ready keeping it sorted.
				idx := sort.SearchStrings(ready, m)
				ready = append(ready, "")
				copy(ready[idx+1:], ready[idx:])
				ready[idx] = m
			}
		}
	}

	// If we couldn't order everything (cycle or malformed upstream), fall back to lexical.
	if len(order) != len(names) {
		fallback := make([]string, len(names))
		copy(fallback, names)
		sort.Strings(fallback)
		return fallback
	}
	return order
}
