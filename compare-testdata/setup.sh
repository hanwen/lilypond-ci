#!/bin/sh

# some simpleminded test data for comparisons
set -eux
TMP=$(mktemp -d )

LILYPOND=${LILYPOND:-lilypond}

if [[ ! -d compare-testdata ]]; then
    echo "run from repo root"
    exit 1
fi

${LILYPOND} -o $TMP -I $PWD/compare-testdata -dcrop -dbackend=cairo --png shift.ly base.ly shift-med.ly shift-large.ly move.ly add.ly remove.ly change.ly  movelarge.ly
(cd $TMP
mkdir d1 d2
mv base.cropped.png d1/base.png
for x in  *.cropped.png ; do
  b=$(basename $x .cropped.png)
  mv $x d2/$b.png
  cp d1/base.png d1/$b.png
done
)

go build -o $TMP/compare ./cmd/compare/
(cd $TMP && ./compare --debug --cmp_jobs=1 --file_regexp '^.*png' $TMP/d1/ $TMP/d2/ $TMP/output/&&
  xdg-open $TMP/output/index.html )
