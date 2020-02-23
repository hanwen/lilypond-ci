#!/bin/sh

if [[ ! -d lilypond ]]; then
    git clone https://git.savannah.gnu.org/git/lilypond.git
else
    (cd lilypond; git fetch origin)
fi

if [[ -z $(docker image list -q lilypond-base-fedora ) ]]; then
    echo "cannot find docker image lilypond-base-fedora"
    echo "run 'setup.sh' first"
    exit 1
fi

docker tag lilypond-base-fedora lilypond-base
docker build -t lilypond-seed-fedora -f lilypond-seed.dockerfile .
docker tag lilypond-base-ubuntu lilypond-base
docker build -t lilypond-seed-ubuntu -f lilypond-seed.dockerfile .
docker tag lilypond-base-fedora-guile2 lilypond-base
docker build -t lilypond-seed-fedora-guile2 -f lilypond-seed.dockerfile .
