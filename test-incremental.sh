#!/bin/bash

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

trap "find /lilypond/ -name '*.fail.log' -exec cp '{}' /output/ ';'" ERR

N=$(nproc)
./autogen.sh --enable-gs-api
time make -j$N
ccache -s

make VERBOSE=1 DESTDIR=/tmp/lp install

if test "${stage}" = build ; then
    exit 0
fi

time make check -j$N CPU_COUNT=$N USE_EXTRACTPDFMARK=no

echo ''
echo ' *** RESULTS ***'
echo ''
cat out/test-results/index.txt

echo ''
echo ' *** CHANGED ***'
echo ''
cat out/test-results/changed.txt

cp -a out/test-results/* /output/

if test "${stage}" != doc ; then
    exit 0
fi

time make doc -j$N CPU_COUNT=$N
