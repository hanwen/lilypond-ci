# This Dockerfile is inspired by the LilyDev docker image,
# https://github.com/fedelibre/LilyDev

FROM ubuntu:16.04

## DEBIAN_FRONTEND=noninteractive prevents apt-get from prompting
## after certain packages are added.
##
## --no-install-recommends avoids installing recommended but not
## required packages, e.g. xterm.
##
## The fonts-texgyre package (the preferred default fonts) is not
## strictly required, since LilyPond can fall back on other fonts, but
## it is convenient to include in the base image so that it is
## consistently available in derived images.
##
RUN apt-get update \
&& DEBIAN_FRONTEND=noninteractive apt-get --no-install-recommends install -y \
    fonts-texgyre \
    ghostscript \
    apt-transport-https \
    ca-certificates \
    guile-1.8 \
    make \
    libpangoft2-1.0-0 \
    python-all \
    python3.5 \
    autoconf \
    autotools-dev \
    bison \
    ccache \
    dblatex \
    debhelper \
    flex \
    fontforge \
    fonts-dejavu \
    fonts-freefont-ttf \
    fonts-ipafont-gothic \
    fonts-ipafont-mincho \
    g++ \
    gdb \
    gettext \
    git \
    groff \
    gsfonts \
    gsfonts-x11 \
    guile-1.8-dev \
    help2man \
    imagemagick \
    less \
    libfl-dev \
    libfontconfig1-dev \
    libfreetype6-dev \
    libgmp3-dev \
    libltdl-dev \
    libpango1.0-dev \
    lmodern \
    m4 \
    make \
    mftrace \
    moreutils \
    netpbm \
    pkg-config \
    quilt \
    rsync \
    strace \
    texi2html \
    texinfo \
    texlive-fonts-recommended \
    texlive-generic-recommended \
    texlive-lang-cyrillic \
    texlive-latex-base \
    texlive-latex-recommended \
    texlive-metapost \
    texlive-xetex \
    tidy \
    zip \
&& rm -rf /var/lib/apt/lists/* /usr/share/doc \
&& /usr/sbin/update-ccache-symlinks
