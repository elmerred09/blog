#!/bin/bash

# lint-on-stop.sh - Run the linter when the agent finishes. If it reports
# problems, ask the agent to keep going and fix them.

input=$(cat)

# Only lint after a normally completed run, not after an abort or error.
status=$(printf '%s' "$input" | sed -n 's/.*"status"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n1)
[ "$status" = "completed" ] || exit 0

[ -f go.mod ] || exit 0

make --no-print-directory lint >/tmp/blog-lint.log 2>&1 && exit 0

# make fails for any reason, including the linter not being installed or
# crashing, which the agent can't fix. Only follow up on real lint findings,
# which golangci-lint reports as "N issues:".
grep -Eq '^[0-9]+ issues?:' /tmp/blog-lint.log || exit 0

# Keep the message static so we never have to JSON-escape lint output.
# The agent reruns the linter itself to see the details.
echo '{"followup_message": "make lint reported problems. Run it, fix the issues it reports, and confirm it passes."}'
exit 0
