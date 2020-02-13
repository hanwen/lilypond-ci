#!/bin/sh

# Usage example:
#
#   sh test-git.sh https://github.com/hanwen/lilypond guile22-experiment
#
# leaves results in
#
#   github.com-hanwen-lilypond/guile22-experiment/COMMIT/
#

set -eu

# Should use gvisor.
runtime=""
if [[ "$(which runsc)" != "" ]]; then
    runtime="--runtime=runsc"
fi

name=$(echo $1 $2 | sed 's|.*://||g;s![/:]!-!g;s| |/|;')
dest="${PWD}/test-results/${name}"
mkdir -p "${dest}"

time docker run "${runtime}" -v ${dest}:/output lilypond-seed /test.sh "$1" "$2"

echo "results in ${dest}"
