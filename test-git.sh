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
command="./autogen.sh && make -j4 && make check CPU_COUNT=4"

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
	    seed_image="lilypond-seed-fedora"
	    command='GUILE_CONFIG=guile-config2.2 GUILE=guile2.2 ./autogen.sh --enable-guile2 && make -j4 && make check CPU_COUNT=4'
	    shift
	    ;;
	*)
	    break 2
	    ;;
    esac
done

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

mkdir -p $
time docker run -v ${dest}:/output \
     -v ${PWD}/lilypond:/${local_repo} \
     -v $HOME/${PWD}/lilypond:/${local_repo} \
     -v ${PWD}/test.sh:/test.sh
     ${seed_image} /test.sh "${url}" "${branch}"


if [[ "$local_repo" = "rietveld" ]]; then
    mv ${dest}/${version} ${dest}/PS${patchset}
fi

echo "results in ${dest}"
