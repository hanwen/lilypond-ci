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

platform=fedora
mode=incremental
while true; do
    case "$1" in
	--ubuntu)
	    platform=ubuntu
	    shift
	    ;;
	--fedora)
	    platform=fedora
	    shift
	    ;;
	--guile2)
	    platform=fedora-guile2
	    shift
	    ;;
	--incr)
	    mode=incremental
	    shift
	    ;;
	--full)
	    mode=full
	    shift
	    ;;
	*)
	    break 2
	    ;;
    esac
done

driver_script=""
if [[ "${mode}" = incremental ]] ; then
    seed_image="lilypond-seed-${platform}"
    driver_script="test-incremental.sh"
else
    seed_image="lilypond-base-${platform}"
    driver_script="test-full.sh"
fi

if [[ -z $(docker image list -q "${seed_image}" ) ]]; then
    echo "cannot find docker image ${seed_image}."
    if [[ "${mode}" = "incremental" ]]; then
	echo "run 'reseed.sh' first"
    else
	echo "run 'setup.sh' first"
    fi
    exit 1
fi

url="$1"
branch="$2"
local_repo="local"

# Test from a branch in a local repo
if [[ -d "${url}" ]] ; then
    url=$(realpath ${url})
    cd lilypond
    # detached head so we can update any branch
    git checkout -f $(git rev-parse origin/master)
    git fetch -f ${url} ${branch}:${branch}
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
dest="${PWD}/../lilypond-test-results/${name}/${seed_image}"
mkdir -p "${dest}"

# TODO: should mount git repo read-only.
time docker run -v ${dest}:/output \
     -v ${PWD}/lilypond:/${local_repo} \
     -v ${PWD}/${driver_script}:/test.sh \
     --rm=true \
     ${seed_image} /test.sh "${url}" "${branch}" "/${local_repo}" "origin/master"

if [[ "$local_repo" = "rietveld" ]]; then
    mv ${dest}/${version} ${dest}/PS${patchset}
fi

echo "results in ${dest}"
