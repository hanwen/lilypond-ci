#!/bin/sh

docker build -t lilypond-base-ubuntu -f ubuntu-xenial.dockerfile .

docker build -t lilypond-base-fedora -f fedora-31.dockerfile .

docker build -t lilypond-base-fedora-guile2 -f fedora-31-guile2.dockerfile .
