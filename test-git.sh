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

base=lilypond-seed-ubuntu
case "$1" in
    --ubuntu)
	base="lilypond-seed-ubuntu"
	shift
	;;
    --fedora)
	base="lilypond-seed-fedora"
	shift
	;;
esac


url="$1"
branch="$2"

if [[ -d "${url}" ]] ; then
    url=$(realpath ${url})
    cd lilypond
    git checkout origin/master
    git fetch -f ${url} ${branch}:${branch}
    local_repo="local"
    url=/local
    cd ..
fi

if [[ "${url}" == "rietveld" ]] ; then
    if [[ ! -d lilypond ]]; then
	git clone https://git.savannah.gnu.org/git/lilypond.git
    fi

    change="$2"
    patchset=$(curl https://codereview.appspot.com/api/${change}/ | jq .patchsets[-1])
    issue="issue${change}_${patchset}"
    cd lilypond
    git fetch origin
    git reset --hard
    git checkout origin/master
    git branch -D ${issue}
    git checkout -b ${issue} origin/master
    curl "https://codereview.appspot.com/download/${issue}.diff" | git apply
    git commit -m "${issue}" -a
    cd ..
    local_repo="rietveld"
    url=/rietveld
    branch=${issue}
fi

name=$(echo $1 $2 | sed 's|.*://||g;s![/:]!-!g;s| |/|;')
dest="${PWD}/test-results/${name}"
mkdir -p "${dest}"

time docker run -v ${dest}:/output \
     -v ${PWD}/lilypond:/${local_repo} \
     lilypond-seed /test.sh "${url}" "${branch}"

echo "results in ${dest}"
