#!/usr/bin/env bash
set -e

SPRINT_NUM="$1"
BASE_DIR="docs/sprints/sprint-${SPRINT_NUM}"
ENGINE=$(grep ENGINE "${BASE_DIR}/sprint-meta.txt" | cut -d '=' -f2)
ENGINE_DIR="${BASE_DIR}/in-process/${ENGINE}-engine"

# Copy planning spec and tdd to implementation folder
cp "${BASE_DIR}/planning/spec.md" "${ENGINE_DIR}/spec.md"
cp "${BASE_DIR}/planning/tdd.md" "${ENGINE_DIR}/tdd.md"

echo "Sprint ${SPRINT_NUM} spec and tdd promoted to implementation folder."
