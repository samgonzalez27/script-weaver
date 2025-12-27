#!/usr/bin/env bash
set -e

SPRINT_NUM="$1"
BASE_DIR="docs/sprints/sprint-${SPRINT_NUM}"
ENGINE=$(grep ENGINE "${BASE_DIR}/sprint-meta.txt" | cut -d '=' -f2)
ENGINE_DIR="${BASE_DIR}/in-process/${ENGINE}-engine"

cat <<EOF
You are the coding agent for sprint ${SPRINT_NUM}, engine ${ENGINE}.

Your task:
- Implement the functionality defined in spec.md
- Write tests and validations from tdd.md
- Log all reasoning, decisions, and blockers in notes.md

Files provided:
- ${ENGINE_DIR}/spec.md
- ${ENGINE_DIR}/tdd.md

Output:
- Code changes in internal/
- notes.md populated with your implementation process

Do not alter spec.md or tdd.md.
EOF
