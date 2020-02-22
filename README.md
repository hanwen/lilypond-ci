Introduction
============

Scripts for testing LilyPond conveniently. It tests LilyPond in the
following 3 configurations:

* Ubuntu Xenial (16.04) with GUILE 1.8. This is represents an "old"
  platform

* Fedora Core 31 with GUILE 1.8. This represents the bleeding edge.

* Fedora Core 31 with GUILE 2.2. This is to ensure that LilyPond keeps
  working against newer GUILE versions.  This requires change
  "Accept GUILE 2 without extra configure options"


Setup
=====

1.  To get started, run `setup.sh` script.  This sets up base images
    for compiling LilyPond.

2.  To create seed images, run `reseed.sh`.  This sets up the a
    regtest baseline.  This should be repeated every time the regtest
    changes significantly

Usage
=====

Start testing (git)

```
# rietveld review
sh test-git.sh rietveld 557410043

# test remote branch on Fedora/guile18
sh test-git.sh --fedora https://github.com/hanwen/lilypond guile22-experiment

# test remote branch on Fedora/guile22
sh test-git.sh --guile2 https://github.com/hanwen/lilypond guile22-experiment

# local branch
sh test-git.sh $HOME/lilypond-src broken-branch
```

This should leave results in `test-results/URL/BRANCH/COMMIT/PLATFORM`
