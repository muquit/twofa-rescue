#!/bin/sh

########################################################################
# Regenerates the fenced usage block in docs/usage.md from the CLI's own
# `-h` output, so the docs never drift from the real flags/examples.
#
# The block to replace is delimited by these marker lines in docs/usage.md:
#   <!-- BEGIN GENERATED USAGE: DO NOT EDIT BY HAND, run scripts/gen_usage.sh -->
#   <!-- END GENERATED USAGE -->
# Everything else in the file (intro prose, the worked "Example:" section
# at the bottom) is left untouched.
########################################################################
set -e

cd "$(dirname "$0")/.."

BINARY=./twofa-rescue
if [ ! -x "$BINARY" ]; then
    echo ">>>> Building $BINARY ..."
    go build -o "$BINARY" .
fi

USAGE_FILE=docs/usage.md
BEGIN_MARK='<!-- BEGIN GENERATED USAGE: DO NOT EDIT BY HAND, run scripts/gen_usage.sh -->'
END_MARK='<!-- END GENERATED USAGE -->'

if [ ! -f "$USAGE_FILE" ]; then
    echo "*** ERROR: $USAGE_FILE not found" >&2
    exit 1
fi

if ! grep -qF "$BEGIN_MARK" "$USAGE_FILE" || ! grep -qF "$END_MARK" "$USAGE_FILE"; then
    echo "*** ERROR: $USAGE_FILE must contain both marker lines:" >&2
    echo "    $BEGIN_MARK" >&2
    echo "    $END_MARK" >&2
    exit 1
fi

USAGE_TMP=$(mktemp)
OUT_TMP=$(mktemp)
trap 'rm -f "$USAGE_TMP" "$OUT_TMP"' EXIT

"$BINARY" -h >"$USAGE_TMP" 2>&1

awk -v begin="$BEGIN_MARK" -v end="$END_MARK" -v usagefile="$USAGE_TMP" '
BEGIN {
    while ((getline line < usagefile) > 0) {
        usage = usage line "\n"
    }
    close(usagefile)
}
$0 == begin {
    print
    print "```"
    printf "%s", usage
    print "```"
    skip = 1
    next
}
$0 == end {
    print
    skip = 0
    next
}
skip { next }
{ print }
' "$USAGE_FILE" >"$OUT_TMP"

mv "$OUT_TMP" "$USAGE_FILE"
echo ">>>> Regenerated usage block in $USAGE_FILE"
