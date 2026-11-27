#!/bin/bash

# guard-push-main.sh - Ask before any git push while main is checked out.
# The permission rules in settings.json allow `git push -u origin HEAD`, which
# can't tell a feature branch from main by pattern alone. A push to main
# deploys to production.

input=$(cat)

printf '%s' "$input" | grep -Eq '"command"[[:space:]]*:[[:space:]]*"[^"]*git[[:space:]]+push' || exit 0

cd "${CLAUDE_PROJECT_DIR:-.}" || exit 0
[ "$(git branch --show-current 2>/dev/null)" = "main" ] || exit 0

echo '{"hookSpecificOutput": {"hookEventName": "PreToolUse", "permissionDecision": "ask", "permissionDecisionReason": "main is checked out: this push would deploy to production."}}'
exit 0
