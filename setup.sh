#!/bin/sh

docker build --no-cache -t lilypond-base-ubuntu -f ubuntu-xenial.dockerfile .

docker build --no-cache -t lilypond-base-fedora -f fedora-31.dockerfile .

docker build --no-cache -t lilypond-base-fedora-guile2 -f fedora-31-guile2.dockerfile .
