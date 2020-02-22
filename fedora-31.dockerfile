from fedora:31 as lilypond-base

RUN dnf update -y
RUN dnf install -y ccache \
  texlive-tex-gyre\
  gcc-c++ t1utils bison flex ImageMagick gettext texlive-tetex\
  python2-devel mftrace texinfo compat-guile18-devel ghostscript\
  pango-devel fontpackages-devel dblatex texinfo-tex texi2html\
  perl-Pod-Parser rsync texlive-tex-gyre texlive-lh texlive-metapost \
  ccache make tidy extractpdfmark autoconf git-core wget perl-Math-Complex \
  automake curl

RUN curl --quiet http://lilypond.org/downloads/gub-sources/texi2html/texi2html-1.82.tar.gz | tar zx \
  && cd texi2html-1.82 \
  && ./configure && make && make install

RUN git clone https://github.com/kohler/t1utils \
  && cd t1utils && ./bootstrap.sh \
  && ./configure && make && make install
