#!/bin/sh

if [[ ! -d lilypond ]]; then
    git clone https://git.savannah.gnu.org/git/lilypond.git
else
    (cd lilypond; git fetch origin)
fi

docker tag lilypond-base-fedora lilypond-base
docker build -t lilypond-seed-fedora -f lilypond-seed.dockerfile .
docker tag lilypond-base-ubuntu lilypond-base
docker build -t lilypond-seed-ubuntu -f lilypond-seed.dockerfile .
docker tag lilypond-base-fedora-guile2 lilypond-base
docker build -t lilypond-seed-fedora-guile2 -f lilypond-seed.dockerfile .
