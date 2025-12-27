#!/usr/bin/env bash
set -e

SPRINT_NUM="$1"
BASE_DIR="docs/sprints/sprint-${SPRINT_NUM}"
ENGINE=$(grep ENGINE "${BASE_DIR}/sprint-meta.txt" | cut -d '=' -f2)
ENGINE_DIR="${BASE_DIR}/in-process/${ENGINE}-engine"

cat <<EOF
You are the documentation agent acting as IMPLEMENTATION REVIEWER for sprint ${SPRINT_NUM}.

Review the implementation notes for engine ${ENGINE}:
- ${ENGINE_DIR}/notes.md

Ensure implementation aligns with:
- ${ENGINE_DIR}/spec.md
- ${ENGINE_DIR}/tdd.md
- Project constraints (docs/project/*)

Return JSON only:
{
  "status": "approved | approved_with_required_changes | not_approved",
  "issues": []
}
EOF
