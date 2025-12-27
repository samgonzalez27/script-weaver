#!/usr/bin/env bash
set -e

SPRINT_NUM="$1"
BASE_DIR="docs/sprints/sprint-${SPRINT_NUM}"

# Move sprint to _complete
mv "${BASE_DIR}" "${BASE_DIR}_complete"

echo "Sprint ${SPRINT_NUM} closed and frozen."
