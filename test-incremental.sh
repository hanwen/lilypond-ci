#!/bin/sh

# checkout a revision and run the tests
# usage. Should run inside the container.
#
#  test.sh STAGE GIT-URL REMOTE-BRANCH

stage=$1
shift

set -eu
cd /lilypond
export PATH="/usr/lib64/ccache:/usr/lib/ccache/:$PATH"

git fetch $1 $2
git checkout FETCH_HEAD

VERSION=$(git rev-parse --short=8 HEAD)
N=$(nproc)
./autogen.sh
time make -j$N
ccache -s

if [[ "${stage}" = build ]] ; then
    exit 0
fi


time make check -j$N CPU_COUNT=$N

echo ''
echo ' *** RESULTS ***'
echo ''
cat out/test-results/index.txt

echo ''
echo ' *** CHANGED ***'
echo ''
cat out/test-results/changed.txt

mkdir -p /output/${VERSION}
cp -a out/test-results/* /output/${VERSION}

if [[ "${stage}" != doc ]] ; then
    exit 0
fi

time make doc -j$N CPU_COUNT=$N
