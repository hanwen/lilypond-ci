#!/bin/sh
set -eu

(cd lilypond && git checkout $1 )

docker run -v $PWD/build:/build -v $PWD/gitlab-ci.sh:/test.sh:ro -v $PWD/lilypond:/lilypond:ro  registry.gitlab.com/lilypond/lilypond/ci/ubuntu-16.04:20200517  /test.sh
