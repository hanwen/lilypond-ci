Introduction
============

These are scripts for testing LilyPond conveniently, using Docker. It
tests LilyPond in the following 3 configurations:

* Ubuntu Xenial (16.04) with GUILE 1.8. This is represents an "old"
  platform

* Fedora Core 31 with GUILE 1.8. This represents the bleeding edge.

* Fedora Core 31 with GUILE 2.2. This is to ensure that LilyPond keeps
  working against newer GUILE versions.


Setup
=====

```
go run test.go --base --platform=ubuntu
go run test.go --seed --platform=ubuntu
```

Other supported platforms are "fedora" (Fedora 31) and "guile2"
(Fedora 31 with Guile 2.2).

Usage
=====

Start testing (git)

```
# rietveld review
go run test.go --test rietveld 557410043

# test remote branch on Fedora/guile18
go run test.go --test --platform=fedora \
  https://github.com/hanwen/lilypond guile22-experiment

# test remote branch on Fedora/guile22
go run test.go --platform=guile2 \
  https://github.com/hanwen/lilypond guile22-experiment

# local branch
go run test.go $HOME/lilypond-src broken-branch
```

This will leave results in `../lilypond/test-results/URL/BRANCH/COMMIT/PLATFORM`
