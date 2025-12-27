#!/usr/bin/env bash
set -e

SPRINT_NUM="$1"
ENGINE_NAME="$2"

BASE_DIR="docs/sprints/sprint-${SPRINT_NUM}"
ENGINE_DIR="${BASE_DIR}/in-process/${ENGINE_NAME}-engine"

# Create directory structure
mkdir -p "${BASE_DIR}"/{backlog,completed,in-process,planning}
mkdir -p "${ENGINE_DIR}"

# Create empty markdown files
touch "${BASE_DIR}/backlog/sprint-${SPRINT_NUM}.md"
touch "${BASE_DIR}/completed/sprint-${SPRINT_NUM}-summary.md"
touch "${ENGINE_DIR}"/{notes.md,tdd.md,spec.md}
touch "${BASE_DIR}/planning"/{planning.md,spec.md,tdd.md,data-dictionary.md}

# Create metadata file for the sprint
cat <<EOF > "${BASE_DIR}/sprint-meta.txt"
ENGINE=${ENGINE_NAME}
EOF

echo "Sprint ${SPRINT_NUM} structure created with engine ${ENGINE_NAME}."
