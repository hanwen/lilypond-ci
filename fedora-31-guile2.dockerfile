from lilypond-base-fedora

RUN dnf install -y guile22-devel guile22 && dnf remove -y compat-guile18-devel compat-guile18 && ln -s guile2.2 /usr/bin/guile
