// Compare a directory of PNG files
package main

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
)

func sqDiffUInt8(x, y uint8) uint64 {
	d := int64(x) - int64(y)
	return uint64(d * d)
}

func sqDiffRGBA(p, q color.RGBA) uint64 {
	r := int64(p.R) - int64(p.R)
	g := int64(p.G) - int64(p.G)
	b := int64(p.B) - int64(p.B)
	a := int64(p.A) - int64(p.A)
	return uint64(r*r) + uint64(g*g) + uint64(b*b) + uint64(a*a)
}

func UnequalCompare(img1, img2 *image.RGBA) (*image.RGBA, float64, error) {
	max := img1.Bounds().Union(img2.Bounds()).Max
	minRect := img1.Bounds().Intersect(img2.Bounds())
	min := minRect.Max

	dst := image.NewRGBA(image.Rectangle{Max: max})
	minXImg := img1
	maxXImg := img2
	if minXImg.Bounds().Max.X > maxXImg.Bounds().Max.X {
		minXImg, maxXImg = maxXImg, minXImg
	}
	minYImg := img1
	maxYImg := img2
	if minYImg.Bounds().Max.Y > maxYImg.Bounds().Max.Y {
		minYImg, maxYImg = maxYImg, minYImg
	}
	white := color.RGBA{0xff, 0xff, 0xff, 0xff}
	var accumError int64
	for y := 0; y < min.Y; y++ {
		for x := 0; x < min.X; x++ {
			diff16 := sqDiffRGBA(img1.RGBAAt(x, y), img2.RGBAAt(x, y))
			accumError += int64(diff16)
			if diff16 > 0 {
				dst.Pix[dst.PixOffset(x, y)] = 0xff
				dst.Pix[dst.PixOffset(x, y)+3] = 0xff
			}
		}
		for x := min.X; x < max.X; x++ {
			diff16 := sqDiffRGBA(maxXImg.RGBAAt(x, y), white)
			accumError += int64(diff16)
			if diff16 > 0 {
				dst.Pix[dst.PixOffset(x, y)] = 0xff
				dst.Pix[dst.PixOffset(x, y)+3] = 0xff
			}
		}
	}
	for y := min.Y; y < max.Y; y++ {
		maxX := maxYImg.Bounds().Max.X
		for x := 0; x < maxX; x++ {
			diff16 := sqDiffRGBA(maxXImg.RGBAAt(x, y), white)
			accumError += int64(diff16)
			if diff16 > 0 {
				dst.Pix[dst.PixOffset(x, y)] = 0xff
				dst.Pix[dst.PixOffset(x, y)+3] = 0xff
			}
		}
	}

	return dst, math.Sqrt(float64(accumError)) / float64(minRect.Dx()*minRect.Dy()), nil
}

func ImageCompare(img1, img2 *image.RGBA) (*image.RGBA, float64, error) {
	if img1.Bounds() != img2.Bounds() {
		return UnequalCompare(img1, img2)
	}

	size := img1.Bounds()
	dst := image.NewRGBA(size)

	accumError := int64(0)

	for i := 0; i < len(img1.Pix); i++ {
		diff16 := sqDiffUInt8(img1.Pix[i], img2.Pix[i])
		accumError += int64(diff16)
		if diff16 > 0 {
			dst.Pix[i-i%4] = 0xff
			dst.Pix[i-i%4+3] = 0xff
		}
	}
	return dst, math.Sqrt(float64(accumError)) / float64(size.Dx()*size.Dy()), nil
}

func readDir(dir string) (map[string]struct{}, error) {
	d, err := filepath.Glob(filepath.Join(flag.Args()[0], "*.png"))
	if err != nil {
		return nil, err
	}
	result := make(map[string]struct{}, len(d))
	for _, n := range d {
		result[filepath.Base(n)] = struct{}{}
	}

	return result, nil
}

func compareDir(in1, in2, out string, ncpu int) error {
	dir1, err := readDir(in1)
	if err != nil {
		return err
	}
	dir2, err := readDir(in2)
	if err != nil {
		return err
	}

	var names []string
	for k := range dir1 {
		if _, ok := dir2[k]; ok {
			names = append(names, k)
		}
	}

	sort.Strings(names)
	type result struct {
		name     string
		dist     float64
		in1, in2 string
		out      string
		err      error
	}
	todo := make(chan *result, len(names))
	done := make(chan *result, len(names))
	for _, k := range names {
		todo <- &result{
			name: k,
			in1:  filepath.Join(in1, k),
			in2:  filepath.Join(in2, k),
			out:  filepath.Join(out, k),
		}
	}
	close(todo)

	for i := 0; i < ncpu; i++ {
		go func() {
			for t := range todo {
				t.dist, t.err = compareOne(t.in1, t.in2, t.out)
				done <- t
			}
		}()
	}

	for range names {
		res := <-done
		log.Printf("compared %s: %f", res.name, res.dist)
		if res.err != nil {
			return res.err
		}
	}

	log.Printf("compared %d images", len(names))
	return nil
}

var (
	ncpu        = flag.Int("jobs", 1, "")
	imageMagick = flag.Bool("imagemagick", false, "")
)

func main() {
	flag.Parse()
	if len(flag.Args()) != 3 {
		log.Fatal("usage: compare input-dir1 input-dir2 output-dir")
	}
	if err := compareDir(flag.Args()[0],
		flag.Args()[1],
		flag.Args()[2], *ncpu); err != nil {
		log.Fatal("compareDir: ", err)
	}
}

func asRGBA(img image.Image) *image.RGBA {
	switch t := img.(type) {

	case *image.NRGBA:
		return &image.RGBA{
			Pix:    t.Pix,
			Stride: t.Stride,
			Rect:   t.Rect,
		}
	case *image.RGBA:
		return t
	default:
		panic("ops")
	}
}

var imageMagickRE = regexp.MustCompile("all: [0-9.e-]* \\(([0-9.e-]*)\\)")

func compareOneIM(in1, in2, out string) (float64, error) {
	cmd := exec.Command("compare", "-verbose", "-metric", "MAE",
		in1, in2, out)

	stdout := &bytes.Buffer{}
	cmd.Stdout = stdout
	cmd.Stderr = stdout

	if err := cmd.Run(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			if ee.ProcessState.ExitCode() == 1 {
				err = nil
			}
		}
		if err != nil {
			return 0, err
		}
	}

	str := stdout.String()

	submatch := imageMagickRE.FindStringSubmatch(str)
	if len(submatch) != 2 {
		return 0, fmt.Errorf("missing re")
	}

	return strconv.ParseFloat(submatch[1], 64)
}

func compareOne(in1, in2, out string) (float64, error) {
	if *imageMagick {
		return compareOneIM(in1, in2, out)
	}

	f1, err := os.Open(in1)
	if err != nil {
		return 0, err
	}
	defer f1.Close()
	i1, err := png.Decode(f1)
	if err != nil {
		return 0, err
	}
	f2, err := os.Open(in2)
	if err != nil {
		return 0, err
	}
	defer f2.Close()
	i2, err := png.Decode(f2)
	if err != nil {
		return 0, err
	}

	diff, dist, err := ImageCompare(asRGBA(i1), asRGBA(i2))
	if err != nil {
		return 0, err
	}
	if dist > 0 {
		os.MkdirAll(filepath.Dir(out), 0755)

		out_f, err := os.OpenFile(out, os.O_WRONLY|os.O_CREATE, 0666)
		if err != nil {
			return 0, err
		}
		if err := png.Encode(out_f, diff); err != nil {
			return 0, err
		}
		return dist, out_f.Close()
	}
	return dist, nil
}
