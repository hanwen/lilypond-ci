
Scripts for testing LilyPond conveniently

Instructions
============

1.  Create the base image, holding all the dev tools

```
docker build -t lilypond-base -f lilypond-base .
```


2.  Create the lilypond seed image, holding the base regression tests

```
docker build -t lilypond-seed -f lilypond-dev .
```

This should be done every time the regression test changes significantly

3.  Start testing (git)

```
sh test-git.sh https://github.com/hanwen/lilypond guile22-experiment
```

This should leave results in

```
github.com-hanwen-lilypond/guile22-experiment/COMMIT/
```

4.  Start testing (Rietveld)

```
sh test-git.sh rietveld 557410043
```
