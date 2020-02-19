#!/bin/sh

docker build -t lilypond-base-ubuntu -f ubuntu-base .
docker build -t lilypond-base-fedora -f fedora-base .

docker tag lilypond-base-fedora lilypond-base
docker build -t lilypond-seed-fedora -f lilypond-seed .
docker tag lilypond-base-ubuntu lilypond-base
docker build -t lilypond-seed-ubuntu -f lilypond-seed .
