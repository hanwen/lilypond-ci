#!/bin/bash

rm -rf /build/*
cd /build
/lilypond/autogen.sh --enable-checking --enable-gs-api --disable-debugging CFLAGS=-O2
N=$(nproc)
MAKE="make -j$N CPU_COUNT=$N"


$MAKE test
$MAKE doc

sleep 20m
