#!/bin/sh

# Usage example:
#
#   sh test-git.sh https://github.com/hanwen/lilypond guile22-experiment
#
# leaves results in
#
#   ../lilypond-test-results/github.com-hanwen-lilypond/guile22-experiment/COMMIT/
#

set -eu

ccache_dir=${HOME}/.cache/lilypond-docker-ccache

seed_image=lilypond-seed-fedora

while true; do
    case "$1" in
	--ubuntu)
	    seed_image="lilypond-seed-ubuntu"
	    shift
	    ;;
	--fedora)
	    seed_image="lilypond-seed-fedora"
	    shift
	    ;;
	--guile2)
	    seed_image="lilypond-seed-fedora-guile2"
	    shift
	    ;;
	*)
	    break 2
	    ;;
    esac
done

url="$1"
branch="$2"

# Test from a branch in a local repo
if [[ -d "${url}" ]] ; then
    url=$(realpath ${url})
    cd lilypond
    git checkout origin/master
    git fetch -f ${url} ${branch}:${branch}
    local_repo="local"
    url=/local
    cd ..
fi

# Test a patch from rietveld
if [[ "${url}" == "rietveld" ]] ; then
    change="$2"
    patchset=$(curl https://codereview.appspot.com/api/${change}/ | jq .patchsets[-1])
    issue="issue${change}_${patchset}"
    cd lilypond
    git fetch origin

    # rebase onto current master
    git checkout -f origin/master
    ( git branch -D ${issue} || true)
    git checkout -b ${issue} origin/master
    curl "https://codereview.appspot.com/download/${issue}.diff" | git apply
    git commit -m "${issue}" -a
    version=$(git rev-parse --short=8 HEAD)
    cd ..
    local_repo="rietveld"
    url=/rietveld
    branch=${issue}
fi

name=$(echo $1 $2 | sed 's|.*://||g;s![/:]!-!g;s| |/|;')
dest="${PWD}/../lilypond-test-results/${name}"
mkdir -p "${dest}"

# TODO: should mount git repo read-only.
time docker run -v ${dest}:/output \
     -v ${PWD}/lilypond:/${local_repo} \
     -v ${PWD}/test.sh:/test.sh \
     ${seed_image} /test.sh "${url}" "${branch}"


if [[ "$local_repo" = "rietveld" ]]; then
    mv ${dest}/${version} ${dest}/PS${patchset}
fi

echo "results in ${dest}"
