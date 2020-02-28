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

for platform in fedora ubuntu fedora-guile2 ; do
    echo ""
    echo "Building seed for ${platform}"
    echo ""

    docker tag lilypond-base-${platform} lilypond-base
    docker build -t lilypond-seed-${platform} -f lilypond-seed.dockerfile .
done
