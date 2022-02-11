from fedora:33 as lilypond-base

COPY init-tex.sh .

RUN dnf update -y && dnf install --setopt=install_weak_deps=False -y \
 ImageMagick \
 autoconf \
 automake \
 bison \
 cairo \
 cairo-devel \
 ccache \
 compat-guile18-devel \
 curl \
 dblatex \
 diffutils \
 dejavu-'*' \
 extractpdfmark \
 flex \
 gdb \
 fontforge \
 fontpackages-devel \
 gcc-c++ \
 gettext \
 ghostscript \
 libgs-devel \
 git-core \
 make \
 pango-devel \
 perl-Math-Complex \
 perl-Pod-Parser \
 rsync \
 t1utils \
 texi2html \
 texinfo \
 texinfo-tex \
 texlive-lh \
 texlive-metapost \
 texlive-tetex \
 texlive-tex-gyre \
 texlive-tex-gyre \
 time \
 && rm -rf /var/cache/dnf \
 && ./init-tex.sh \
 && curl --silent http://lilypond.org/downloads/gub-sources/texi2html/texi2html-1.82.tar.gz | tar zx \
 && cd texi2html-1.82 \
 && ./configure && make && make install \
 && cd .. \
 && git clone https://github.com/kohler/t1utils \
 && cd t1utils && ./bootstrap.sh \
 && ./configure && make && make install

# t1utils is fubar on Fedora. See https://github.com/kohler/t1utils/issues/8 and
# https://bugzilla.redhat.com/show_bug.cgi?id=1777987

# lilypond requires a very specific texi2html version.
