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

echo "Checking dependencies..."
if [ ! -d "node_modules" ]; then
    echo "Installing Playwright dependencies..."
    npm install
fi

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
