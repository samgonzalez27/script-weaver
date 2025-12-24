# Sprint-08 Backlog & Known Limitations

**Generated**: 2025-12-23

## Deferred Work
*   **Advanced Resume Selection**: Currently, `incremental` mode auto-selects the most recent failed run. Explicit selection of *any* past run requires CLI arguments not yet designed.
*   **Partial Resume/Cherry-Picking**: Resuming only a subgraph is out of scope. Resume is currently all-or-nothing for the graph.

## Known Limitations
*   **JSON Scalability**: Execution state runs are stored as individual JSON files. Very high-frequency runs (millions) may strain the filesystem (inode usage). This is acceptable for current scale goals.
*   **Node ID constraints**: Resume persistence assumes Node IDs are safe filenames. Graphs with complex characters in Node IDs may fail to persist checkpoints (detected by strict validation).
*   **Run ID Format**: Run IDs are currently random 128-bit hex strings. This is an implementation detail, not a hard contract, but consumers rely on it being unique.

## Future Considerations
*   **Artifact GC**: We track runs indefinitely. A future sprint (Cleanup Policies) must address garbage collection of old `.scriptweaver/runs/` and cache entries.
*   **Distributed Resume**: Current lock + atomic file semantics work for single-machine execution. Distributed execution will require a dedicated state store (Redis/SQL).
