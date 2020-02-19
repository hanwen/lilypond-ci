
Scripts for testing LilyPond conveniently

Setup
=====

Run the `setup.sh` script. Or:

1.  Create the base image, holding all the dev tools

```
docker build -t lilypond-base -f ubuntu-base .  # or use 'fedora-base'
```


2.  Create the lilypond seed image, holding the base regression tests

```
docker build -t lilypond-seed -f lilypond-seed .
```

This should be done every time the regression test changes significantly


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
