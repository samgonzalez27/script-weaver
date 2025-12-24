# Execution State Data Dictionary

## `.scriptweaver/runs/<run-id>/`

* ### `run.json`
    * run_id
    * graph_hash
    * start_time
    * mode
    * retry_count
    * status (running | failed | completed)
    * previous_run_id (nullable)
* ### `checkpoint.json`
    * node_id
    * timestamp
    * cache_keys
    * output_hash
    * valid (boolean)
* ### `failure.json` (only if failure)
    * failure_class
    * node_id (if applicable)
    * error_code
    * error_message
    * resumable (boolean)

## `.scriptweaver/cache/`

* Immutable per graph hash
* Cache entries must be checksum-verified before reuse
* Missing or corrupted cache entries invalidate downstream checkpoints

## Invariants (Non-Negotiable)

* Resume never skips required work
* Resume never replays completed work unless invalidated
* Resume never mutates user project files
* Failure state is explicit, inspectable, and durable
