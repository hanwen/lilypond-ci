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

trap 'cp $(find /lilypond/ -name "*.fail.log") /output/' EXIT

N=$(nproc)
./autogen.sh  --enable-gs-api
export PATH="/usr/lib64/ccache:/usr/lib/ccache/:$PATH"

case "${stage}" in
    doc|check)
	time make -j$N
	time make test-baseline -j$N CPU_COUNT=$N
	make distclean
	;;
esac

git fetch $1 $2:test
git checkout test
./autogen.sh
time make -j$N

case "${stage}" in
build)
    exit 0
    ;;
doc)
    time make doc -j$N CPU_COUNT=$N USE_EXTRACTPDFMARK=no
    ;;
esac

time make check -j$N CPU_COUNT=$N USE_EXTRACTPDFMARK=no

echo ''
echo ' *** RESULTS ***'
echo ''
cat out/test-results/index.txt

cp -a out/test-results/* /output/
