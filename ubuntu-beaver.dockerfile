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

FROM ubuntu:18.04 as build

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
# Download and build Guile 1.8.8, there's no package in Ubuntu 18.04.
RUN wget -q https://ftp.gnu.org/gnu/guile/guile-1.8.8.tar.gz \
    && tar xf guile-1.8.8.tar.gz && mkdir build-guile1.8 && cd build-guile1.8 \
    && /guile-1.8.8/configure --prefix=/usr --disable-error-on-warning \
    && make -j$(nproc) && make install-strip DESTDIR=/install-guile1.8

FROM ubuntu:18.04
COPY init-tex.sh .
COPY --from=build /usr/share/fonts/otf/ /usr/share/fonts/otf/
COPY --from=build /install-guile1.8/ /
RUN apt-get update && apt-get --no-install-recommends install -y \
        autoconf \
        bison \
	ccache \
        ca-certificates \
        flex \
        fontconfig \
        fontforge \
        fonts-texgyre \
        g++ \
        gettext \
        ghostscript \
        git \
        imagemagick \
	libcairo-dev \
        libfl-dev \
        libfontconfig1-dev \
        libfreetype6-dev \
        libglib2.0-dev \
        libgmp-dev \
        libgs-dev \
        libltdl-dev \
        libpango1.0-dev \
        make \
        perl \
        pkg-config \
        python3 \
        rsync \
        texi2html \
        texinfo \
        texlive-binaries \
        texlive-fonts-recommended \
        texlive-lang-cyrillic \
        texlive-metapost \
        texlive-plain-generic \
	texlive-latex-base \
	texlive-latex-recommended \
	texlive-xetex \
        wget \
        zip \
    && rm -rf /var/lib/apt/lists/* \
    && rm -rf /usr/share/doc /usr/share/man \
    && /usr/sbin/update-ccache-symlinks \
    && ./init-tex.sh

