from fedora:31 as lilypond-base

RUN dnf update -y && dnf install --setopt=install_weak_deps=False -y \
  ccache texlive-tex-gyre fontforge \
  gcc-c++ t1utils bison flex ImageMagick gettext texlive-tetex \
  texinfo compat-guile18-devel ghostscript \
  pango-devel fontpackages-devel dblatex texinfo-tex texi2html \
  perl-Pod-Parser rsync texlive-tex-gyre texlive-lh texlive-metapost \
  ccache make tidy extractpdfmark autoconf git-core perl-Math-Complex \
  automake curl \
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
