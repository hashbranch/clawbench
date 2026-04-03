#!/usr/bin/env bash
# Validates email triage classification from stdin.
# Expected: 8 numbered items classified as action/todo/noise.
# Minimum 7/8 correct to pass.
set -euo pipefail

RESPONSE=$(cat)

# Expected classifications (lowercase)
# 1=CRM notification(noise) 2=attorney(action) 3=restaurant promo(noise)
# 4=cofounder(action) 5=GitHub bot(noise) 6=client contract(action)
# 7=newsletter(noise) 8=prospect assistant(action)
declare -A EXPECTED=(
  [1]="noise"
  [2]="action"
  [3]="noise"
  [4]="action"
  [5]="noise"
  [6]="action"
  [7]="noise"
  [8]="action"
)

correct=0
total=8

for i in $(seq 1 $total); do
  # Extract classification for item number $i
  # Match patterns like "1. noise", "1: noise", "1 - noise", "1) noise"
  line=$(echo "$RESPONSE" | grep -iE "^[[:space:]]*${i}[.):,\-][[:space:]]" | head -1 || true)
  if [ -z "$line" ]; then
    # Try looser match — just the number anywhere on line
    line=$(echo "$RESPONSE" | grep -iE "(^|[[:space:]])${i}[.):,\-]" | head -1 || true)
  fi

  expected="${EXPECTED[$i]}"

  if echo "$line" | grep -qi "$expected"; then
    ((correct++))
  else
    echo "FAIL item $i: expected=$expected, got line: $line" >&2
  fi
done

echo "Score: $correct/$total" >&2

if [ "$correct" -ge 7 ]; then
  exit 0
else
  echo "FAIL: Only $correct/$total correct (need 7+)" >&2
  exit 1
fi
