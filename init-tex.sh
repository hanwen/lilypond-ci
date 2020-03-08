#!/bin/sh

set -eu
tmp=$(mktemp -d)
cd $tmp
cat <<EOF > dummy-latex.tex
\documentclass{article}
\begin{document}
hoi
\end{document}
EOF
cat <<EOF > dummy-tex.tex
hello
\bye
EOF

pdflatex dummy-latex.tex
xelatex  dummy-latex.tex

tex dummy-tex
etex dummy-tex
pdfetex dummy-tex
pdftex dummy-tex
xetex dummy-tex
