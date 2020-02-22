#!/bin/sh

if [[ ! -d lilypond ]]; then
    git clone https://git.savannah.gnu.org/git/lilypond.git
else
    (cd lilypond; git fetch origin)
fi

# TODO - should work version into image tag name?
REV=origin/master
eval $(git --git-dir lilypond/.git show ${REV}:VERSION)
LILY_VERSION=${MAJOR_VERSION}.${MINOR_VERSION}.${PATCH_LEVEL}
VERSION=$(date --iso-8601=minutes | tr -d ':' | tr '[A-Z]' '[a-z]' \
	      | sed 's|\+.*$||')-${LILY_VERSION}-$(git --git-dir liypond rev-parse --short origin/master)


docker tag lilypond-base-fedora lilypond-base
docker build -t lilypond-seed-fedora -f lilypond-seed .
docker tag lilypond-base-ubuntu lilypond-base
docker build -t lilypond-seed-ubuntu -f lilypond-seed .
docker tag lilypond-base-fedora-guile2 lilypond-base
docker build -t lilypond-seed-fedora-guile2 -f lilypond-seed .
