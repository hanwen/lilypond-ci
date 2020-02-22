Introduction
============


Scripts for testing LilyPond conveniently. It tests LilyPond in the
following 3 configurations:

* Ubuntu Xenial (16.04) with GUILE 1.8. This is represents an "old"
  platform

* Fedora Core 31 with GUILE 1.8. This represents the bleeding edge.

* Fedora Core 31 with GUILE 2.2. This is to ensure that LilyPond keeps
  working against newer GUILE versions

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
# remote branch
sh --fedora test-git.sh https://github.com/hanwen/lilypond guile22-experiment

# local branch
sh --ubuntu test-git.sh $HOME/lilypond-src broken-branch

# rietveld review
sh test-git.sh rietveld 557410043
```

This should leave results in `test-results/URL/BRANCH/COMMIT`
