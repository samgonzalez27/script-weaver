#!/usr/bin/env bash
set -e

SPRINT_NUM="$1"
BASE_DIR="docs/sprints/sprint-${SPRINT_NUM}"
ENGINE=$(grep ENGINE "${BASE_DIR}/sprint-meta.txt" | cut -d '=' -f2)
ENGINE_DIR="${BASE_DIR}/in-process/${ENGINE}-engine"

cat <<EOF
You are the documentation agent responsible for generating the CLOSURE content for sprint ${SPRINT_NUM}.

Your task:
- Populate the following files based on the sprint implementation:
  - Backlog: ${BASE_DIR}/backlog/sprint-${SPRINT_NUM}.md
  - Completed summary: ${BASE_DIR}/completed/sprint-${SPRINT_NUM}-summary.md

Available context:
- Planning docs: ${BASE_DIR}/planning/*
- Implementation notes: ${ENGINE_DIR}/notes.md
- Project constraints: docs/project/*

Instructions:
1. In backlog/sprint-${SPRINT_NUM}.md, list:
   - Deferred or unfinished features
   - Known limitations
   - Considerations or recommendations for future sprints

2. In completed/sprint-${SPRINT_NUM}-summary.md, summarize:
   - Sprint goals
   - Key implementation challenges
   - Outcomes achieved
   - Lessons learned

Output:
- Content suitable to be directly placed into the respective markdown files
- Avoid adding unrelated analysis or commentary

EOF
