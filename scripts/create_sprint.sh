#!/usr/bin/env bash
set -e

SPRINT_NUM="$1"
ENGINE_NAME="$2"

BASE_DIR="docs/sprints/sprint-${SPRINT_NUM}"

mkdir -p "${BASE_DIR}"/{backlog,completed,in-process,planning}

touch "${BASE_DIR}/backlog/sprint-${SPRINT_NUM}.md"
touch "${BASE_DIR}/completed/sprint-${SPRINT_NUM}-summary.md"

ENGINE_DIR="${BASE_DIR}/in-process/${ENGINE_NAME}-engine"
mkdir -p "${ENGINE_DIR}"
touch "${ENGINE_DIR}"/{notes.md,tdd.md,spec.md}

touch "${BASE_DIR}/planning"/{planning.md,spec.md,tdd.md,data-dictionary.md}

echo "Sprint ${SPRINT_NUM} structure created."
