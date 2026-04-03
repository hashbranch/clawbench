#!/usr/bin/env bash
# Validates professional email tone from stdin.
# Checks for AI-isms, sycophantic openers, word count, CTA, buzzwords.
set -euo pipefail

RESPONSE=$(cat)
errors=0

# 1. No em dashes (— U+2014, – U+2013)
if echo "$RESPONSE" | grep -qP '[\x{2014}\x{2013}]'; then
  echo "FAIL: Contains em dash or en dash" >&2
  ((errors++))
fi

# 2. No sycophantic openers
if echo "$RESPONSE" | grep -qiE "(I'd be happy to|Great to|^Absolutely)"; then
  echo "FAIL: Contains sycophantic opener" >&2
  ((errors++))
fi

# 3. No generic filler
if echo "$RESPONSE" | grep -qi "I hope this email finds you well"; then
  echo "FAIL: Contains generic filler" >&2
  ((errors++))
fi

# 4. Under 100 words (count words in the email body, not the whole response)
word_count=$(echo "$RESPONSE" | wc -w | tr -d ' ')
if [ "$word_count" -gt 150 ]; then
  # Allow some slack for the agent's framing around the email
  echo "FAIL: Too many words ($word_count, max ~150 including framing)" >&2
  ((errors++))
fi

# 5. Contains a clear CTA
if ! echo "$RESPONSE" | grep -qiE "(schedule|call|meeting|chat|connect|book|set up|hop on)"; then
  echo "FAIL: No clear CTA (call/meeting/schedule/etc.)" >&2
  ((errors++))
fi

# 6. No excessive exclamation marks (more than 2 lines with !)
excl_lines=$(echo "$RESPONSE" | grep -c '!' || true)
if [ "$excl_lines" -gt 2 ]; then
  echo "FAIL: Too many exclamation marks ($excl_lines lines)" >&2
  ((errors++))
fi

# 7. No buzzwords
if echo "$RESPONSE" | grep -qiE "\b(leverage|synergy|synergies)\b"; then
  echo "FAIL: Contains buzzwords (leverage/synergy)" >&2
  ((errors++))
fi

if [ "$errors" -gt 0 ]; then
  echo "FAIL: $errors violations found" >&2
  exit 1
fi

echo "PASS: Professional tone checks passed" >&2
exit 0
