#!/bin/bash

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

trap "find /lilypond/ -name '*.fail.log' -exec cp '{}' /output/ ';'" ERR

export PATH="/usr/lib64/ccache:/usr/lib/ccache/:$PATH"
/lilypond/autogen.sh  --enable-gs-api

case "${stage}" in
    doc|check)
	time make -j$N
	time make test-baseline -j$N CPU_COUNT=$N USE_EXTRACTPDFMARK=no
	;;
    *)
	make distclean
	;;
esac

cd /lilypond
git fetch $1 $2:test
git checkout test

cd /lpbuild
/lilypond/autogen.sh  --enable-gs-api

time make -j$N
time make DESTDIR=/tmp/lp install

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
