#!/usr/bin/env bash

# runner.sh - Agent-friendly script to execute UI tests headlessly.
#
# Usage:
#   ./runner.sh [spec-file.ts]
#
# Examples:
#   ./runner.sh                  # Runs all tests
#   ./runner.sh basic.spec.ts    # Runs a specific test file

set -e

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$DIR"

BASE_URL="${UI_BASE_URL:-http://localhost:34115}"
WAIT_TIMEOUT_SECONDS="${UI_WAIT_TIMEOUT_SECONDS:-60}"

echo "Checking dependencies..."
if [ ! -d "node_modules" ] || ! node -e "require('@playwright/test')" >/dev/null 2>&1; then
    echo "Installing Playwright dependencies from package-lock.json..."
    npm ci
fi

if ! node -e "const fs = require('fs'); const { chromium } = require('@playwright/test'); fs.accessSync(chromium.executablePath())" >/dev/null 2>&1; then
    echo "Installing Playwright Chromium..."
    npx playwright install chromium
fi

echo "Waiting for Wails dev at $BASE_URL..."
deadline=$((SECONDS + WAIT_TIMEOUT_SECONDS))
until curl -fsS "$BASE_URL" >/dev/null 2>&1; do
    if [ "$SECONDS" -ge "$deadline" ]; then
        echo "Wails dev did not become ready within ${WAIT_TIMEOUT_SECONDS}s."
        echo "Start it from the project root with: wails dev"
        exit 1
    fi
    sleep 1
done

echo "Wails dev is ready."

# Determine if the user passed a specific spec file
SPEC_FILE=$1
if [ -z "$SPEC_FILE" ]; then
    echo "Running all Playwright tests headlessly..."
    npx playwright test
else
    echo "Running specific Playwright test: $SPEC_FILE headlessly..."
    npx playwright test "$SPEC_FILE"
fi

echo ""
echo "Tests completed! Results saved to test-results/"
if [ -f "test-results/report.json" ]; then
    echo "A JSON report is available for agents to parse: test-results/report.json"
fi
