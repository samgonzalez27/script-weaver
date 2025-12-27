#!/usr/bin/env bash
set -e

SPRINT_NUM="$1"
BASE_DIR="docs/sprints/sprint-${SPRINT_NUM}"
PROJECT_DIR="docs/project"

ENGINE=$(grep ENGINE "${BASE_DIR}/sprint-meta.txt" | cut -d '=' -f2)

cat <<EOF
You are the documentation agent acting as PLANNING AUTHORITY for sprint ${SPRINT_NUM}.

Your task is to REVIEW the planning documents and ensure:
- coherence
- completeness
- alignment with project constraints

Files to review:
- ${BASE_DIR}/planning/planning.md
- ${BASE_DIR}/planning/spec.md
- ${BASE_DIR}/planning/tdd.md
- ${BASE_DIR}/planning/data-dictionary.md

GLOBAL PROJECT CONTEXT:
- ${PROJECT_DIR}/vision.md
- ${PROJECT_DIR}/constraints.md
- ${PROJECT_DIR}/architecture.md

CURRENT SPRINT CONTEXT:
- Implementation engine: ${ENGINE}

EOF

if [[ "${SPRINT_NUM}" != "00" && -f "docs/sprints/sprint-${PREV_SPRINT}_complete/backlog/sprint-${PREV_SPRINT}.md" ]]; then
cat <<EOF
PREVIOUS SPRINT BACKLOG:
- docs/sprints/sprint-${PREV_SPRINT}_complete/backlog/sprint-${PREV_SPRINT}.md
EOF
fi

cat <<'EOF'

Return JSON only:

{
  "status": "approved | approved_with_clarifications | rejected",
  "issues": []
}
EOF
