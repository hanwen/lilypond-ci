# This file is part of LilyPond, the GNU music typesetter.
#
# Copyright (C) 2020--2022  Jonas Hahnfeld <hahnjo@hahnjo.de>
#
# LilyPond is free software: you can redistribute it and/or modify
# it under the terms of the GNU General Public License as published by
# the Free Software Foundation, either version 3 of the License, or
# (at your option) any later version.
#
# LilyPond is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU General Public License for more details.
#
# You should have received a copy of the GNU General Public License
# along with LilyPond.  If not, see <http://www.gnu.org/licenses/>.

FROM ubuntu:22.04 as build

RUN apt-get update && apt-get install --no-install-recommends -y \
        binutils \
        ca-certificates \
	file \
        gcc \
        libc-dev \
        libgmp-dev \
        libltdl-dev \
        make \
        wget \
    && true
# Download and extract urw-base35-fonts - we cannot use the available package
# fonts-urw-base35 which comes without the *.otf files. Do this in a build
# container to not distribute the whole archive, only the *.otf files.
# Do not use ADD which doesn't cache the downloaded archive. Once released, it
# will never change.
RUN wget -q https://github.com/ArtifexSoftware/urw-base35-fonts/archive/20170801.1.tar.gz \
    && mkdir -p /usr/share/fonts/otf/ && tar -C /usr/share/fonts/otf/ \
        -xf /20170801.1.tar.gz --strip-components=2 --wildcards '*/fonts/*.otf'

FROM ubuntu:22.04
COPY --from=build /usr/share/fonts/otf/ /usr/share/fonts/otf/
RUN apt-get update && \
	DEBIAN_FRONTEND=noninteractive apt-get --no-install-recommends install -y \
	binutils-mingw-w64-x86-64 \ 
	ccache \
	mingw-w64-x86-64-dev \
	mingw-w64 \
	mingw-w64-common \
	mingw-w64-tools \
	g++-mingw-w64-x86-64 \
	gcc-mingw-w64-x86-64 \
	gperf \
	less \
	libfl-dev \
	meson \
	mingw-w64-tools \
	texlive-latex-base \
	texlive-latex-recommended \
	texlive-xetex \
        autoconf \
        bison \
        ca-certificates \
        flex \
        fontforge \
        fonts-texgyre \
        g++ \
        gettext \
        git \
        imagemagick \
	icoutils \
        make \
        perl \
	pkgconf \
        python3 \
        rsync \
        texi2html \
        texinfo \
        texlive-binaries \
        texlive-fonts-recommended \
        texlive-lang-cyrillic \
        texlive-metapost \
        texlive-plain-generic \
        wget \
        zip \
    && /usr/sbin/update-ccache-symlinks 
    

