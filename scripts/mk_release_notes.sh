#!/usr/bin/env bash
#
# Generate Markdown release notes from release archives in bin/.
#
# The script is intended to be copied between Go projects without editing.
# It derives the program name from go.mod, the version from VERSION (or the
# current Git tag), and the repository URL from the Git remote.
#
# Optional overrides:
#   PROG=name VERSION=v1.2.3 BIN_DIR=dist OUT_FILE=notes.md \
#     VERSION_FILE=VERSION REPO_URL=https://example.com/owner/repo \
#     ./scripts/mk_release_notes.sh

set -euo pipefail

die() {
    echo "*** ERROR: $*" >&2
    exit 1
}

command_exists() {
    command -v "$1" >/dev/null 2>&1
}

archive_os() {
    local archive_name target goos
    archive_name=${1##*/}
    target=${archive_name#"$PROG-$VERSION-"}
    goos=${target%%-*}

    case "$goos" in
        darwin) echo "macOS" ;;
        windows) echo "Windows" ;;
        linux) echo "Linux" ;;
        freebsd) echo "FreeBSD" ;;
        openbsd) echo "OpenBSD" ;;
        netbsd) echo "NetBSD" ;;
        dragonfly) echo "DragonFly BSD" ;;
        solaris) echo "Solaris" ;;
        android) echo "Android" ;;
        ios) echo "iOS" ;;
        aix) echo "AIX" ;;
        plan9) echo "Plan 9" ;;
        *) echo "$goos" ;;
    esac
}

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
repo_dir=$(CDPATH= cd -- "$script_dir/.." && pwd)
cd "$repo_dir"

bin_dir=${BIN_DIR:-bin}
version_file=${VERSION_FILE:-VERSION}
out_file=${OUT_FILE:-release_notes.md}

if [ -z "${PROG:-}" ]; then
    if [ -f go.mod ]; then
        module=$(awk '$1 == "module" { print $2; exit }' go.mod)
        PROG=${module##*/}
    else
        PROG=${repo_dir##*/}
    fi
fi
[ -n "$PROG" ] || die "could not determine the program name; set PROG"

if [ -z "${VERSION:-}" ]; then
    if [ -f "$version_file" ]; then
        IFS= read -r VERSION < "$version_file" || true
    elif command_exists git; then
        VERSION=$(git describe --tags --exact-match 2>/dev/null || true)
    fi
fi
[ -n "${VERSION:-}" ] || die "could not determine the version; create $version_file, run from a tagged commit, or set VERSION"

if [ -z "${REPO_URL:-}" ] && command_exists git; then
    REPO_URL=$(git remote get-url origin 2>/dev/null || true)
    case "$REPO_URL" in
        git@*:* )
            remote=${REPO_URL#git@}
            host=${remote%%:*}
            path=${remote#*:}
            REPO_URL="https://$host/$path"
            ;;
        ssh://git@* ) REPO_URL="https://${REPO_URL#ssh://git@}" ;;
    esac
    REPO_URL=${REPO_URL%.git}
fi

[ -d "$bin_dir" ] || die "$bin_dir not found; build the release archives first"

shopt -s nullglob
archives=(
    "$bin_dir/$PROG-$VERSION-"*.tar.gz
    "$bin_dir/$PROG-$VERSION-"*.tgz
    "$bin_dir/$PROG-$VERSION-"*.zip
)
shopt -u nullglob
[ ${#archives[@]} -gt 0 ] || die "no $PROG-$VERSION release archives found in $bin_dir"

for archive in "${archives[@]}"; do
    case "$archive" in
        *.zip) command_exists unzip || die "unzip is required to inspect $archive" ;;
        *.tar.gz|*.tgz) command_exists tar || die "tar is required to inspect $archive" ;;
    esac
done

checksum_file=""
for candidate in \
    "$bin_dir/$PROG-$VERSION-checksums.txt" \
    "$bin_dir/checksums.txt"; do
    if [ -f "$candidate" ]; then
        checksum_file=$candidate
        break
    fi
done

{
    echo "# Release $VERSION"
    echo

    if [ -f ChangeLog.md ]; then
        echo "See [ChangeLog.md](ChangeLog.md) for details about this release."
        echo
    elif [ -f CHANGELOG.md ]; then
        echo "See [CHANGELOG.md](CHANGELOG.md) for details about this release."
        echo
    fi

    echo "## Downloads"
    echo
    echo "| OS | Archive | Size (bytes) |"
    echo "|---|---|---:|"
    for archive in "${archives[@]}"; do
        name=${archive##*/}
        os=$(archive_os "$archive")
        size=$(wc -c < "$archive" | tr -d '[:space:]')
        if [ -n "${REPO_URL:-}" ]; then
            echo "| $os | [$name]($REPO_URL/releases/download/$VERSION/$name) | $size |"
        else
            echo "| $os | $name | $size |"
        fi
    done
    echo

    echo "## Installation"
    echo
    echo "Download and extract the archive for your platform, then copy the"
    echo "executable to a directory in your \`PATH\`. You may rename it to"
    echo "\`$PROG\` (or \`$PROG.exe\` on Windows)."
    echo

    if [ -n "$checksum_file" ]; then
        echo "## Checksums"
        echo
        echo '```text'
        sed 's| \*\?bin/| |' "$checksum_file"
        echo '```'
        echo
    fi

    echo "## Archive contents"
    for archive in "${archives[@]}"; do
        name=${archive##*/}
        echo
        echo "### $name"
        echo
        echo '```text'
        case "$archive" in
            *.zip) unzip -Z1 "$archive" ;;
            *.tar.gz|*.tgz) tar -tzf "$archive" ;;
        esac
        echo '```'
    done

    if [ -f README.md ]; then
        echo
        echo "See [README.md](README.md) for complete usage and build instructions."
    fi
    echo
    echo "Cross-compiled and released with [go-xbuild-go](https://github.com/muquit/go-xbuild-go)."
} > "$out_file"

echo ">>>> Wrote $out_file for $PROG $VERSION (${#archives[@]} archives)"
