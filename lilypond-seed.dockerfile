from lilypond-base

# need 2 ccache paths for both Ubuntu and Fedora
ENV PATH /usr/lib/ccache:/usr/lib64/ccache/:$PATH

# Since we're building an image, can't use bind mounts. Copy the repo instead.
WORKDIR /
RUN mkdir /lilypond
COPY lilypond/.git /lilypond/.git
WORKDIR /lilypond
RUN git checkout -f origin/master

RUN ./autogen.sh && make -j$(nproc) \
  && make test-baseline -j$(nproc) CPU_COUNT=$(nproc) \
  && make distclean \
  && ccache -z
