#!/bin/sh

########################################################################
# Wraps an iPhone screenshot in a real photographic device frame using
# fastlane's `frameit`, instead of the hand-drawn bezel in
# scripts/iphone_frame.sh. Frames are matched automatically by the
# screenshot's pixel dimensions, so this only works for resolutions
# frameit recognizes (real, unedited iPhone screenshots) -- it will
# fail or skip a file it can't match. For arbitrary/resized images,
# use scripts/iphone_frame.sh instead.
#
# Requirements:
#   - fastlane (`fastlane` command)   gem install fastlane
#   - device frame assets, downloaded once and cached in
#     ~/.fastlane/frameit/latest -- this script downloads them
#     automatically on first run if missing
#
# Usage:
#   scripts/iphone_frame_frameit.sh screenshot.png
#       -> writes screenshot_framed.png (black frame, frameit's default)
#   scripts/iphone_frame_frameit.sh -w shot1.png shot2.png ...
#       -> white frames, several files at once
#   scripts/iphone_frame_frameit.sh -o outdir screenshot.png
#       -> writes outdir/screenshot_framed.png
#
#   Color flags (pick at most one): -w white, -g gold, -r rose gold,
#   -s silver
########################################################################
set -e

usage() {
    echo "Usage: $0 [-w|-g|-r|-s] [-o outdir] screenshot.png [screenshot2.png ...]" >&2
    echo "  -w          white device frame" >&2
    echo "  -g          gold device frame" >&2
    echo "  -r          rose gold device frame" >&2
    echo "  -s          silver device frame" >&2
    echo "  -o outdir   write framed files into outdir instead of alongside each input" >&2
    exit 1
}

if ! command -v fastlane >/dev/null 2>&1; then
    echo "*** ERROR: 'fastlane' command not found. Install with: gem install fastlane" >&2
    exit 1
fi

COLOR_FLAG=""
OUTDIR=""

while getopts "wgrso:" opt; do
    case "$opt" in
        w) COLOR_FLAG="--white" ;;
        g) COLOR_FLAG="--gold" ;;
        r) COLOR_FLAG="--rose_gold" ;;
        s) COLOR_FLAG="--silver" ;;
        o) OUTDIR="$OPTARG" ;;
        *) usage ;;
    esac
done
shift $((OPTIND - 1))

if [ "$#" -eq 0 ]; then
    usage
fi

for IN in "$@"; do
    if [ ! -f "$IN" ]; then
        echo "*** ERROR: $IN not found" >&2
        exit 1
    fi
done

if [ ! -d "$HOME/.fastlane/frameit/latest" ]; then
    echo ">>>> Downloading device frame assets (one-time) ..."
    fastlane frameit download_frames
fi

if [ -n "$OUTDIR" ]; then
    mkdir -p "$OUTDIR"
fi

WORKDIR=$(mktemp -d)
trap 'rm -rf "$WORKDIR"' EXIT

for IN in "$@"; do
    cp "$IN" "$WORKDIR/"
done

(cd "$WORKDIR" && fastlane frameit $COLOR_FLAG)

for IN in "$@"; do
    BASE=$(basename "$IN" | sed -E 's/\.[^.]+$//')
    FRAMED=$(ls "$WORKDIR/${BASE}_framed."* 2>/dev/null | head -1)

    if [ -z "$FRAMED" ]; then
        echo "*** WARNING: frameit did not produce a framed image for $IN (unrecognized resolution?)" >&2
        continue
    fi

    if [ -n "$OUTDIR" ]; then
        DEST="$OUTDIR/$(basename "$FRAMED")"
    else
        DEST="$(dirname "$IN")/$(basename "$FRAMED")"
    fi

    mv "$FRAMED" "$DEST"
    echo ">>>> wrote $DEST"
done
