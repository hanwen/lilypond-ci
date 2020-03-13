#!/bin/sh

# checkout a base revision, create baseline, checkout rev to test, run
# the tests, make doc.
#
#  test.sh STAGE GIT-URL REMOTE-BRANCH  LOCAL-GIT-DIRECTORY LOCAL-BASELINE

stage=$1
shift

set -eu

mkdir /lilypond
cd /lilypond
cp -a $3/.git .
git checkout -f $4

N=$(nproc)

mkdir /lpbuild
cd /lpbuild
export PATH="/usr/lib64/ccache:/usr/lib/ccache/:$PATH"
/lilypond/autogen.sh

case "${stage}" in
    doc|check)
	time make -j$N
	time make test-baseline -j$N CPU_COUNT=$N
	make distclean
	;;
esac

cd /lilypond
git fetch $1 $2:test
git checkout test
VERSION=$(git rev-parse --short=8 HEAD)

cd /lpbuild
/lilypond/autogen.sh

time make -j$N
if [[ "${stage}" = build ]] ; then
    exit 0
fi

if [[ "${stage}" = doc ]] ; then
    time make doc -j$N CPU_COUNT=$N
fi

time make check -j$N CPU_COUNT=$N

echo ''
echo ' *** RESULTS ***'
echo ''
cat out/test-results/index.txt

mkdir -p /output/${VERSION}
cp -a out/test-results/* /output/${VERSION}
