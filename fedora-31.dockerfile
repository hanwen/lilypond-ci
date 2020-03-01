from fedora:31 as lilypond-base

RUN dnf update -y && dnf install --setopt=install_weak_deps=False -y \
 ImageMagick \
 autoconf \
 automake \
 bison \
 ccache \
 compat-guile18-devel \
 curl \
 dblatex \
 dejavu-'*' \
 extractpdfmark \
 flex \
 fontforge \
 fontpackages-devel \
 gcc-c++ \
 gettext \
 ghostscript \
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
 tidy \
 time \
 && rm -rf /var/cache/dnf

# lilypond requires a very specific texi2html version.
RUN curl --silent http://lilypond.org/downloads/gub-sources/texi2html/texi2html-1.82.tar.gz | tar zx \
  && cd texi2html-1.82 \
  && ./configure && make && make install

# t1utils is fubar on Fedora. See https://github.com/kohler/t1utils/issues/8 and
# https://bugzilla.redhat.com/show_bug.cgi?id=1777987
RUN git clone https://github.com/kohler/t1utils \
  && cd t1utils && ./bootstrap.sh \
  && ./configure && make && make install
