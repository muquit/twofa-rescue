#!/bin/sh

########################################################################
# Wraps an iPhone screenshot in a black bezel with rounded corners and a
# Dynamic Island cutout, for use in README/docs screenshots.
#
# Requirements:
#   - ImageMagick 7+ (`magick` command)   brew install imagemagick
#   - awk, sed, mktemp                    standard on macOS/Linux
#

# Usage:
#   scripts/iphone_frame.sh screenshot.png
#       -> writes screenshot-framed.png
#   scripts/iphone_frame.sh -o out.png screenshot.png
#       -> writes out.png (single input only)
#   scripts/iphone_frame.sh -i shot1.png shot2.png ...
#       -> overwrites each input in place
#
# All bezel dimensions (border, corner radius, island size) are scaled
# from the input image's own width, so this works on any iPhone
# screenshot resolution (portrait, full-height, no letterboxing).
########################################################################
set -e

usage() {
    echo "Usage: $0 [-i] [-o outfile] screenshot.png [screenshot2.png ...]" >&2
    echo "  -i          overwrite each input file in place" >&2
    echo "  -o outfile  write to outfile (only valid with a single input)" >&2
    exit 1
}

if ! command -v magick >/dev/null 2>&1; then
    echo "*** ERROR: ImageMagick 'magick' command not found. Install with: brew install imagemagick" >&2
    exit 1
fi

IN_PLACE=0
OUTFILE=""

while getopts "io:" opt; do
    case "$opt" in
        i) IN_PLACE=1 ;;
        o) OUTFILE="$OPTARG" ;;
        *) usage ;;
    esac
done
shift $((OPTIND - 1))

if [ "$#" -eq 0 ]; then
    usage
fi

if [ -n "$OUTFILE" ] && [ "$IN_PLACE" -eq 1 ]; then
    echo "*** ERROR: -i and -o are mutually exclusive" >&2
    exit 1
fi

if [ -n "$OUTFILE" ] && [ "$#" -gt 1 ]; then
    echo "*** ERROR: -o only works with a single input file" >&2
    exit 1
fi

frame_one() {
    IN="$1"
    OUT="$2"

    if [ ! -f "$IN" ]; then
        echo "*** ERROR: $IN not found" >&2
        return 1
    fi

    W=$(magick identify -format "%w" "$IN")
    H=$(magick identify -format "%h" "$IN")

    # Bezel proportions derived from a reference 1206px-wide iPhone
    # screenshot, scaled to whatever width this image actually is.
    SCREEN_RADIUS=$(awk -v w="$W" 'BEGIN { printf "%d", w * 0.1161 }')
    BORDER=$(awk -v w="$W" 'BEGIN { printf "%d", w * 0.0365 }')
    ISL_W=$(awk -v w="$W" 'BEGIN { printf "%d", w * 0.2786 }')
    ISL_H=$(awk -v w="$W" 'BEGIN { printf "%d", w * 0.0813 }')
    ISL_TOP_GAP=$(awk -v w="$W" 'BEGIN { printf "%d", w * 0.0415 }')
    STROKE=$(awk -v w="$W" 'BEGIN { s = int(w * 0.0017); print (s < 1 ? 1 : s) }')

    OUTER_RADIUS=$((SCREEN_RADIUS + BORDER))
    CW=$((W + 2 * BORDER))
    CH=$((H + 2 * BORDER))
    ISL_X=$(( (CW - ISL_W) / 2 ))
    ISL_Y=$((BORDER + ISL_TOP_GAP))
    ISL_RADIUS=$((ISL_H / 2))

    TMP=$(mktemp -d)
    trap 'rm -rf "$TMP"' EXIT

    # round the screenshot's own corners
    magick -size "${W}x${H}" xc:black -fill white \
        -draw "roundrectangle 0,0,$((W - 1)),$((H - 1)),${SCREEN_RADIUS},${SCREEN_RADIUS}" \
        "$TMP/mask.png"
    magick "$IN" "$TMP/mask.png" -alpha off -compose CopyOpacity -composite "$TMP/rounded_screen.png"

    # bezel canvas with a faint highlight ring
    magick -size "${CW}x${CH}" xc:none -fill "#111113" \
        -draw "roundrectangle 0,0,$((CW - 1)),$((CH - 1)),${OUTER_RADIUS},${OUTER_RADIUS}" \
        -fill none -stroke "#3a3a3d" -strokewidth "$STROKE" \
        -draw "roundrectangle 2,2,$((CW - 3)),$((CH - 3)),${OUTER_RADIUS},${OUTER_RADIUS}" \
        "$TMP/bezel.png"

    # composite screen into bezel, then draw the Dynamic Island on top
    magick "$TMP/bezel.png" "$TMP/rounded_screen.png" -geometry "+${BORDER}+${BORDER}" -compose over -composite \
        -fill black -stroke "#2c2c2e" -strokewidth "$STROKE" \
        -draw "roundrectangle ${ISL_X},${ISL_Y},$((ISL_X + ISL_W)),$((ISL_Y + ISL_H)),${ISL_RADIUS},${ISL_RADIUS}" \
        "$OUT"

    rm -rf "$TMP"
    trap - EXIT
    echo ">>>> wrote $OUT"
}

for IN in "$@"; do
    if [ -n "$OUTFILE" ]; then
        DEST="$OUTFILE"
    elif [ "$IN_PLACE" -eq 1 ]; then
        DEST="$IN"
    else
        BASE=$(dirname "$IN")/$(basename "$IN" | sed -E 's/\.[^.]+$//')
        EXT=$(basename "$IN" | sed -E 's/^.*\.([^.]+)$/\1/')
        DEST="${BASE}-framed.${EXT}"
    fi

    if [ "$IN_PLACE" -eq 1 ]; then
        TMPOUT="${IN}.tmp.$$.png"
        frame_one "$IN" "$TMPOUT"
        mv "$TMPOUT" "$DEST"
    else
        frame_one "$IN" "$DEST"
    fi
done
