#!/bin/sh

# checkout a base revision, create baseline, checkout rev to test, run
# the tests, make doc.
#
#  test.sh GIT-URL REMOTE-BRANCH  LOCAL-GIT-DIRECTORY LOCAL-BASELINE

set -eu

mkdir /lilypond
cd /lilypond
cp -a $3/.git .
git checkout -f $4

N=$(nproc)
./autogen.sh
export PATH="/usr/lib64/ccache:/usr/lib/ccache/:$PATH"
time make -j$(nproc)
time make test-baseline -j$N CPU_COUNT=$N
make dist-clean

git fetch $1 $2:test
git checkout test
time make -j$N
time make doc -j$N CPU_COUNT=$N
time make check -j$N CPU_COUNT=$N

echo ''
echo ' *** RESULTS ***'
echo ''
cat out/test-results/index.txt

mkdir -p /output/${VERSION}
cp -a out/test-results/* /output/${VERSION}