#!/usr/bin/env bash
set -e

SPRINT_NUM="$1"
BASE_DIR="docs/sprints/sprint-${SPRINT_NUM}"
PROJECT_DIR="docs/project"

# Read engine name from sprint-meta.txt
ENGINE=$(grep ENGINE "${BASE_DIR}/sprint-meta.txt" | cut -d '=' -f2)

PREV_SPRINT=$((SPRINT_NUM - 1))
PREV_BACKLOG="docs/sprints/sprint-${PREV_SPRINT}_complete/backlog/sprint-${PREV_SPRINT}.md"

cat <<EOF
You are the documentation agent acting as PLANNING AUTHOR for sprint ${SPRINT_NUM}.

Your task is to CREATE the following files:
- ${BASE_DIR}/planning/planning.md
- ${BASE_DIR}/planning/spec.md
- ${BASE_DIR}/planning/tdd.md
- ${BASE_DIR}/planning/data-dictionary.md

Populate these files with complete, coherent content using the context below.

GLOBAL PROJECT CONTEXT (authoritative, read-only):
- ${PROJECT_DIR}/vision.md
- ${PROJECT_DIR}/constraints.md
- ${PROJECT_DIR}/architecture.md

CURRENT SPRINT CONTEXT:
- Sprint number: ${SPRINT_NUM}
- Implementation engine: ${ENGINE}

EOF

if [[ "${SPRINT_NUM}" != "00" && -f "${PREV_BACKLOG}" ]]; then
cat <<EOF
PREVIOUS SPRINT BACKLOG (historical context):
- ${PREV_BACKLOG}

Incorporate backlog items or explicitly defer them into the current planning docs.
EOF
fi

cat <<'EOF'

Guidelines for writing the planning documents:
- Define the sprint goal clearly in planning.md
- Specify feature requirements and behavior in spec.md
- Describe test scenarios and validation in tdd.md
- Record relevant data structures and meanings in data-dictionary.md
- Make assumptions and constraints explicit
- Avoid hallucinations; content must be based on provided context
- Output content suitable to be directly placed into the corresponding markdown files

Do not perform review or validation. Focus solely on creation.
EOF
