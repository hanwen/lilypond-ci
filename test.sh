#!/bin/sh

# checkout a revision and run the tests
# usage. Should run inside the container.
#
#  test.sh GIT-URL REMOTE-BRANCH

set -eu
cd /lilypond
export PATH="/usr/lib64/ccache/:$PATH"

git fetch $1 $2
git checkout FETCH_HEAD

VERSION=$(git rev-parse --short=8 HEAD)

./autogen.sh
make -j4

ccache -s

make check CPU_COUNT=4

echo ''
echo ' *** RESULTS ***'
echo ''
cat out/test-results/index.txt

echo ''
echo ' *** CHANGED ***'
echo ''
cat out/test-results/changed.txt

mkdir /output/${VERSION}
cp -a out/test-results/* /output/${VERSION}
