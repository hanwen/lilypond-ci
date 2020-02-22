#!/bin/sh

docker build -t lilypond-base-ubuntu -f ubuntu-xenial-base .

docker build -t lilypond-base-fedora -f fedora-base .

docker build -t lilypond-base-fedora-guile2 -f fedora-guile2-base .
