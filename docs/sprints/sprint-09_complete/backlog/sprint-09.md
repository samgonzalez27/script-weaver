# Sprint-09 Backlog & Deferred Items

## Deferred Features

### Dynamic Runtime Loading
**Context**: The sprint plan specified loading "external Go modules" but did not define the ABI or symbol resolution mechanism (e.g., `plugin` package vs. distinct processes).
**Status**: Deferred. The current implementation provides the *structure* (discovery, manifest parsing, hook registry, execution loop) but relies on in-memory registration for now.
**Future Work**: Define a strict ABI or go-plugin interface for loading compiled artifacts.

### Explicit Mutation Tests
**Context**: TDD called for "Plugin cannot mutate forbidden state".
**Status**: Satisfied by construction (read-only interfaces).
**Future Work**: If mutable context is added in future sprints, explicit test cases must be added to verify access control.

## Known Limitations

*   **No CLI Management**: There are no `scriptweaver plugin list/install` commands. Management is manual (filesystem operations).
*   **Synchronous Execution**: All hooks are synchronous; slow plugins will block the core engine.

## Future Opportunities (No Commitment)

*   **Plugin CLI**: deeply integrated commands for managing local plugins.
*   **Remote Distribution**: "Marketplace" or git-based plugin fetching.
*   **Rich Context**: Exposing safe, controlled mutability (e.g., modifying specific task metadata) if use cases demand it.
