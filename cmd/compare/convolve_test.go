package main

import (
	"image"
	"testing"
)

func TestSumImage(t *testing.T) {
	r := image.Rectangle{
		Max: image.Point{2, 2},
	}
	in := newSignedImage(r)
	/*
	   1 2
	   3 4
	*/

	in.Set(0, 0, 1.0)
	in.Set(1, 0, 2.0)
	in.Set(0, 1, 3.0)
	in.Set(1, 1, 4.0)

	ii := newIntegralImage(in)
	want := map[image.Point]float64{
		image.Point{0, 0}:  1,
		image.Point{-1, 0}: 0,
		image.Point{2, 0}:  3,
		image.Point{1, 1}:  10.0,
		image.Point{2, 1}:  10.0,
		image.Point{2, 2}:  10.0,
		image.Point{0, 2}:  4.0,
	}

	for k, wantVal := range want {
		if got := ii.Val(k.X, k.Y); got != wantVal {
			t.Errorf("%v: got %f want %f", k, got, wantVal)
		}
	}
}
