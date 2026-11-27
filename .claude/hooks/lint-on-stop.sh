#!/bin/bash

# lint-on-stop.sh - Run the linter when Claude finishes. If it reports
# problems, block the stop and ask Claude to keep going and fix them.

input=$(cat)

# Claude Code sets stop_hook_active when Claude is already continuing because
# of this hook. Let it stop then, so a finding it can't fix doesn't loop.
printf '%s' "$input" | grep -Eq '"stop_hook_active"[[:space:]]*:[[:space:]]*true' && exit 0

cd "${CLAUDE_PROJECT_DIR:-.}" || exit 0
[ -f go.mod ] || exit 0

log=$(mktemp)
trap 'rm -f "$log"' EXIT

make --no-print-directory lint >"$log" 2>&1 && exit 0

# make fails for any reason, including the linter not being installed or
# crashing, which Claude can't fix. Only follow up on real lint findings,
# which golangci-lint reports as "N issues:".
grep -Eq '^[0-9]+ issues?:' "$log" || exit 0

# Keep the message static so we never have to JSON-escape lint output.
# Claude reruns the linter itself to see the details.
echo '{"decision": "block", "reason": "make lint reported problems. Run it, fix the issues it reports, and confirm it passes."}'
exit 0
