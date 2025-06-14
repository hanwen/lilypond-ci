// Compare a directory of PNG files
package main

import (
	"bytes"
	"flag"
	"fmt"
	"html/template"
	"image"
	"image/color"
	"image/png"
	"io"
	"log"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

func absdiff(a, b uint8) uint32 {
	if a > b {
		return uint32(a - b)
	}
	return uint32(b - a)
}

func sqDiffUInt8(x, y uint8) (uint64, uint32) {
	d := absdiff(x, y)
	return uint64(d * d), d
}

func sqDiffRGBA(p, q color.RGBA) (uint64, uint8) {
	r := absdiff(p.R, q.R)
	g := absdiff(p.G, q.G)
	b := absdiff(p.B, q.B)

	// ignore alpha
	return uint64(r*r + g*g + b*b), uint8((r + g + b) / 3)
}

type whiteBGImage struct {
	*image.RGBA
}

var white = color.RGBA{
	0xff, 0xff, 0xff, 0xff,
}

func (i *whiteBGImage) RGBAAt(x, y int) color.RGBA {
	if x < i.Rect.Min.X || x >= i.Rect.Max.X {
		return white
	}
	if y < i.Rect.Min.Y || y >= i.Rect.Max.Y {
		return white
	}

	return i.RGBA.RGBAAt(x, y)
}

type signedImage struct {
	Pix    []float64
	Stride int
	Rect   image.Rectangle
}

func newSignedImage(rect image.Rectangle) *signedImage {
	return &signedImage{
		Rect:   rect,
		Stride: rect.Dx(),
		Pix:    make([]float64, rect.Dx()*rect.Dy()),
	}
}

type sumImage struct {
	*signedImage
}

func (p *sumImage) Val(x, y int) float64 {
	if y < p.Rect.Min.Y {
		return 0.0
	}
	if x < p.Rect.Min.X {
		return 0.0
	}
	if x >= p.Rect.Max.X {
		x = p.Rect.Max.X - 1
	}
	if y >= p.Rect.Max.Y {
		y = p.Rect.Max.Y - 1
	}

	return p.signedImage.Val(x, y)
}

func (p *signedImage) Val(x, y int) float64 {
	if x < p.Rect.Min.X || x >= p.Rect.Max.X {
		return 0.0
	}
	if y < p.Rect.Min.Y || y >= p.Rect.Max.Y {
		return 0.0
	}
	return p.Pix[p.PixOffset(x, y)]
}

func (p *signedImage) Set(x, y int, v float64) {
	p.Pix[p.PixOffset(x, y)] = v
}

func (p *signedImage) PixOffset(x, y int) int {
	return (y-p.Rect.Min.Y)*p.Stride + (x - p.Rect.Min.X)
}

func (p *signedImage) AvgAbs() float64 {
	sum := 0.0
	for _, v := range p.Pix {
		sum += math.Abs(v)
	}
	return sum / float64(len(p.Pix))
}

func (p *signedImage) dump(fn string) error {
	r := p.Rect

	img := image.NewRGBA(r)
	for y := range r.Max.Y {
		for x := range r.Max.X {
			v := p.Val(x, y)
			img.SetRGBA(x, y, unitToRGBA(v))
		}
	}

	outF, err := os.OpenFile(fn, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	if err := png.Encode(outF, img); err != nil {
		return err
	}
	return outF.Close()
}

func rgbSum(c color.RGBA) float64 {
	return float64(c.R) + float64(c.G) + float64(c.B)
}

func newIntegralImage(in *signedImage) *sumImage {
	s := &sumImage{newSignedImage(in.Rect)}
	for y := range in.Rect.Max.Y {
		for x := range in.Rect.Max.X {
			s.Set(x, y, in.Val(x, y)+s.Val(x-1, y)+s.Val(x, y-1)-s.Val(x-1, y-1))
		}
	}
	return s
}

func ImageCompareConvolve(img1, img2 *image.RGBA, name string) (*image.RGBA, float64, error) {
	maxPoint := img1.Bounds().Union(img2.Bounds()).Max
	maxRect := image.Rectangle{Max: maxPoint}
	diffImg := newSignedImage(maxRect)

	w1 := whiteBGImage{img1}
	w2 := whiteBGImage{img2}

	for y := range maxPoint.Y {
		for x := range maxPoint.X {
			c1 := w1.RGBAAt(x, y)
			c2 := w2.RGBAAt(x, y)

			// diff has range (-1,1)
			diff := (rgbSum(c1) - rgbSum(c2)) / (3 * 255.0)

			diffImg.Set(x, y, diff)
		}
	}

	sumDiffImg := newIntegralImage(diffImg)
	resolutions := []*signedImage{diffImg}
	delta := 1
	for delta < maxRect.Dx() && delta < maxRect.Dy() {
		dst := newSignedImage(maxRect)
		for y := range maxPoint.Y {
			for x := range maxPoint.X {
				boxSum := sumDiffImg.Val(x+delta, y+delta) - sumDiffImg.Val(x+delta, y-delta-1) +
					sumDiffImg.Val(x-delta-1, y-delta-1) - sumDiffImg.Val(x-delta-1, y+delta)
				width := float64(2*delta + 1)
				dst.Set(x, y, boxSum/(width*width))
			}
		}

		resolutions = append(resolutions, dst)
		delta *= 2
	}

	diffRGB := image.NewRGBA(maxRect)
	if *debug {
		for i, r := range resolutions {
			log.Printf("%s: %d: %e", name, i, r.AvgAbs())
			r.dump(fmt.Sprintf("%s-%d.png", name, i))
		}
	}

	dist := 0.0
	for y := range maxPoint.Y {
		for x := range maxPoint.X {
			diff := 0.0

			for _, r := range resolutions {
				d := r.Val(x, y)
				diff += d
			}

			diff /= float64(len(resolutions))

			if diff > 1 || diff < -1 {
				log.Fatalf("range %f", diff)
			}
			dist += math.Abs(diff)

			// -1,1
			diffRGB.SetRGBA(x, y, unitToRGBA(diff))
		}
	}

	// Normalize by image size.
	dist /= float64(maxPoint.X * maxPoint.Y)

	return diffRGB, dist, nil
}

func ImageCompareMAE(img1, img2 *image.RGBA) (*image.RGBA, float64, error) {
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
			sqDiff, absDiff := sqDiffRGBA(img1.RGBAAt(x, y), img2.RGBAAt(x, y))
			accumError += int64(sqDiff)
			if absDiff > 0 {
				dst.Pix[dst.PixOffset(x, y)] = 0xff
				dst.Pix[dst.PixOffset(x, y)+3] = absDiff
			}
		}
		for x := min.X; x < max.X; x++ {
			sqDiff, absDiff := sqDiffRGBA(maxXImg.RGBAAt(x, y), white)
			accumError += int64(sqDiff)
			if absDiff > 0 {
				dst.Pix[dst.PixOffset(x, y)] = 0xff
				dst.Pix[dst.PixOffset(x, y)+3] = absDiff
			}
		}
	}
	for y := min.Y; y < max.Y; y++ {
		maxX := maxYImg.Bounds().Max.X
		for x := 0; x < maxX; x++ {
			sqDiff, absDiff := sqDiffRGBA(maxXImg.RGBAAt(x, y), white)
			accumError += int64(sqDiff)
			if absDiff > 0 {
				dst.Pix[dst.PixOffset(x, y)] = 0xff
				dst.Pix[dst.PixOffset(x, y)+3] = absDiff
			}
		}
	}

	return dst, math.Sqrt(float64(accumError)) / float64(minRect.Dx()*minRect.Dy()), nil
}

func unitToRGBA(diff float64) (pix color.RGBA) {
	green := color.RGBA{G: 0xff}
	red := color.RGBA{R: 0xff}
	if diff < 0 {
		pix = green
	} else {
		pix = red
	}
	d := math.Abs(diff)
	if d > 1.0 {
		log.Printf("diff %f", d)
		d = 1.0
	}

	// Nudge nonzero values towards 1, so they are more pronounced.
	d = math.Pow(d, 0.5)
	pix.A = uint8(d * 0xff)
	return
}

var debug = flag.Bool("debug", false, "")
var batchGS = flag.Bool("batch_gs", true, "")
var localDataDir = flag.Bool("local", false, "")
var verbose = flag.Bool("verbose", false, "")

func convertEPSParallel(eps_files map[string]string, ncpu int) error {
	sz := len(eps_files) / ncpu
	if sz == 0 {
		sz++
	}

	chunks := make([]map[string]string, ncpu)
	for i := range chunks {
		chunks[i] = make(map[string]string)
	}
	i := 0
	for k, v := range eps_files {
		chunks[i%ncpu][k] = v
		i++
	}

	type chunkResult struct {
		filemap map[string]string
		err     error
	}
	done := make(chan chunkResult, ncpu)
	for _, chunk := range chunks {
		go func(ch map[string]string) {
			var r chunkResult
			r.err = convertEPS(ch)
			done <- r
		}(chunk)
	}

	result := map[string]string{}
	for range chunks {
		r := <-done
		if r.err != nil {
			return r.err
		}

		for k, v := range r.filemap {
			result[k] = v
		}
	}
	return nil
}

func EPSBBoxEmpty(fn string) (bool, error) {
	f, err := os.Open(fn)
	if err != nil {
		return false, err
	}
	defer f.Close()

	buf := make([]byte, 1024)
	n, err := f.Read(buf)
	if err != nil {
		return false, err
	}

	header := string(buf[:n])

	marker := "\n%%BoundingBox: "
	idx := strings.Index(header, marker)
	if idx < 0 {
		return false, fmt.Errorf("no bbox in %s", fn)
	}

	header = header[idx+len(marker):]
	header = header[:strings.Index(header, "\n")]
	var dims []int
	for _, n := range strings.Split(header, " ") {
		dim, err := strconv.Atoi(n)
		if err != nil {
			return false, err
		}
		dims = append(dims, dim)
	}

	return dims[0] >= dims[2] || dims[1] >= dims[3], nil
}

func convertEPS(epsFiles map[string]string) error {
	if *batchGS {
		return convertEPSBatch(epsFiles)
	}

	for k, v := range epsFiles {
		if err := convertEPSBatch(map[string]string{k: v}); err != nil {
			return err
		}
	}

	return nil
}

func convertEPSBatch(epsFiles map[string]string) error {
	if len(epsFiles) == 0 {
		return nil
	}
	dataOption := ""
	if *localDataDir {
		doneDir := map[string]bool{}
		for fn := range epsFiles {
			dir := filepath.Dir(fn)

			if doneDir[dir] {
				continue
			}
			doneDir[dir] = true
			fi, err := os.Stat(filepath.Join(dir, "share"))
			if err != nil && os.IsNotExist(err) {
				continue
			}
			if err != nil {
				return err
			}
			dir, err = filepath.Abs(dir)
			if err != nil {
				return err
			}
			if fi.IsDir() {
				dataOption = fmt.Sprintf("-slilypond-datadir=%s/share/lilypond/current", dir)
				break
			}
		}
	}

	emptyPS, err := os.CreateTemp("", "emptyps")
	if err != nil {
		return err
	}
	if err := os.WriteFile(emptyPS.Name(), []byte(`%!PS-Adobe-3.0 EPSF-3.0
%%BoundingBox: 0 0 1 1
%%EndComments
`), 0644); err != nil {
		return err
	}
	emptyPS.Close()
	driver, err := os.CreateTemp("", "driverps")
	if err != nil {
		return err
	}

	for inputFn, outFn := range epsFiles {
		verbosePS := ""
		if *verbose {
			verbosePS = fmt.Sprintf(" (processing %s\n) print ", inputFn)
		}

		if empty, err := EPSBBoxEmpty(inputFn); err != nil {
			return fmt.Errorf("EPSBBoxEmpty: %v", err)
		} else if empty {
			inputFn = emptyPS.Name()
		}

		_, err = fmt.Fprintf(driver, `
            %s
            mark /OutputFile (%s)
            /GraphicsAlphaBits 4 /TextAlphaBits 4
            /HWResolution [101 101]
            (png16m) finddevice putdeviceprops setdevice
            (%s) run
`, verbosePS, outFn, inputFn)
		if err != nil {
			return err
		}
	}

	if err := driver.Close(); err != nil {
		return err
	}
	cmd := exec.Command(
		"gs",
		"-dNOSAFER",
		"-dEPSCrop",
		"-q",
		"-dNOPAUSE",
		"-dNODISPLAY",
		"-dAutoRotatePages=/None",
		"-dPrinted=false")
	if dataOption != "" {
		cmd.Args = append(cmd.Args, dataOption)
	}
	cmd.Args = append(cmd.Args, driver.Name(), "-c", "quit")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if *verbose {
		log.Printf("calling %v", cmd.Args)
	}
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func compareDir(in1, in2 string, fileRegexp string) (*compareResult, error) {
	re, err := regexp.Compile(fileRegexp)
	if err != nil {
		return nil, err
	}

	in1 = filepath.Clean(in1)
	in2 = filepath.Clean(in2)
	res := map[string]*fileResult{}
	for i, dir := range []string{in1, in2} {
		fns, err := os.ReadDir(dir)
		if err != nil {
			return nil, err
		}

		for _, fn := range fns {
			name := fn.Name()
			if re.FindString(name) == "" {
				continue
			}

			name = filepath.Base(name)
			name = name[:len(name)-len(filepath.Ext(name))]

			fr := res[name]
			if fr == nil {
				fr = &fileResult{Name: name}
				res[name] = fr
			}

			if !strings.HasSuffix(fr.In[i], ".png") {
				fr.In[i] = filepath.Join(dir, fn.Name())
			}
		}
	}

	return &compareResult{
		byName: res,
		dirs:   [2]string{in1, in2},
	}, nil
}

func (r *compareResult) renderPNG(outDir string) error {
	start := time.Now()
	epsFileCount := 0

	for i := 0; i < 2; i++ {
		fnMap := map[string]string{}
		for _, v := range r.byName {
			if strings.HasSuffix(v.In[i], ".eps") {
				newname := filepath.Join(outDir, fmt.Sprintf("%s.%d.png", v.Name, i))
				fnMap[v.In[i]] = newname
				v.In[i] = newname
			}
		}

		epsFileCount += len(fnMap)
		if err := convertEPSParallel(fnMap, *gsJobs); err != nil {
			return err
		}
	}
	epsDT := time.Now().Sub(start)
	log.Printf("Convert %d EPS files using %d cores (batch=%v) to PNG in %v (%v/file)", epsFileCount, *gsJobs, *batchGS, epsDT, epsDT/(1+time.Duration(epsFileCount)))
	return nil
}

type compareResult struct {
	byName  map[string]*fileResult
	dirs    [2]string
	Results []*fileResult
}

func (r *compareResult) Trim(max int) {
	r.Results = nil
	for _, v := range r.byName {
		r.Results = append(r.Results, v)
	}

	sort.Slice(r.Results, func(i, j int) bool { return r.Results[i].Dist > r.Results[j].Dist })
	for i := range r.Results {
		if r.Results[i].Dist == 0.0 || (max > 0 && i > max) {
			r.Results = r.Results[:i]
			break
		}
	}
}

func (r *compareResult) LinkFiles(outDir string) error {
	for _, r := range r.Results {
		for i := 0; i < 2; i++ {
			if strings.HasPrefix(r.In[i], outDir) {
				continue
			}
			if err := os.Link(r.In[i], filepath.Join(outDir, fmt.Sprintf("%s.%d.png", r.Name, i))); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *compareResult) DumpTXT(w io.Writer) {
	for _, r := range r.Results {
		fmt.Fprintf(w, "%-50s - %f\n", r.Name, r.Dist)
	}
}

type fileResult struct {
	Name    string
	Dist    float64
	DistMAE float64
	In      [2]string
	out     string
	err     error
}

var htmlTemplate *template.Template

func init() {
	htmlTemplate = template.Must(template.New("html").Parse(`
<html>
  <style>
    table, th, td {
      border: 1px solid grey;
    }
  </style>
  <title>Image comparison</title>
  <body>
    <table>
      <tr><th>dist</th><th>old</th><th>new</th><th>MAE</th></tr>
      {{range .Results}}
         {{template "entry" .}}        
      {{end}}
    </table>
  </body>
</html>
`))
	template.Must(htmlTemplate.New("entry").Parse(`
<tr>
  <td>
    {{printf "%.4f" .Dist}}
  </td>
  <td>
    <div>
      <div style="position: absolute">
	<img src="{{.Name}}.0.png">
      </div>
      <div style="opacity: 0.0">
         <img src="{{.Name}}.diff.png">
      </div>
    <div>
    <br>
    {{.Name}}
  </td>
  <td>
    <div>
      <div style="position: absolute">
         <img src="{{.Name}}.1.png">
      </div>
      <div style="position: absolute; opacity: 1.0">
         <img src="{{.Name}}.diff.png">
      </div>
      <div style="opacity: 0.0">
         <img src="{{.Name}}.diff.png">
      </div>
    </div>
    <br>
    {{.Name}}
  </td>
  <td>
    {{printf "%.4f" .DistMAE}}
  </td>
</tr>
`))
}

func (r *compareResult) DumpHTMLFile(outDir string) error {
	f, err := os.Create(filepath.Join(outDir, "index.html"))
	if err != nil {
		return err
	}

	if err := r.DumpHTML(f); err != nil {
		return err
	}
	return f.Close()
}

func (r *compareResult) DumpHTML(w io.Writer) error {
	return htmlTemplate.Execute(w, r)
}

func (r *compareResult) comparePNG(outDir string, ncpu int) error {
	start := time.Now()

	todo := make(chan *fileResult, len(r.byName))
	scheduled := 0
	for _, v := range r.byName {
		if v.In[0] != "" && v.In[1] != "" {
			v.out = filepath.Join(outDir, v.Name+".diff.png")
			scheduled++
			todo <- v
		}
	}
	close(todo)
	done := make(chan *fileResult, scheduled)
	for i := 0; i < ncpu; i++ {
		go func() {
			for t := range todo {
				t.err = t.compareOne()
				done <- t
			}
		}()
	}

	for i := 0; i < scheduled; i++ {
		<-done
	}

	pngDT := time.Now().Sub(start)
	log.Printf("compared %d PNG image pairs using %d cores (imagemagick=%v) in %v (%v / pair)", scheduled, ncpu, *imageMagick, pngDT, pngDT/time.Duration(1+scheduled))

	for _, fr := range r.byName {
		if fr.err != nil {
			return fr.err
		}
	}
	return nil
}

var (
	gsJobs      = flag.Int("gs_jobs", runtime.NumCPU(), "")
	cmpJobs     = flag.Int("cmp_jobs", runtime.NumCPU(), "")
	imageMagick = flag.Bool("imagemagick", false, "")
	maxOutputs  = flag.Int("max", 0, "output top-N differences")
)

func main() {
	fileRegexp := flag.String("file_regexp", "-[0-9][0-9]*.(eps|png)$",
		"compare only files matching this regexp")
	flag.Parse()
	if len(flag.Args()) != 3 {
		log.Fatal("usage: compare input-dir1 input-dir2 output-dir")
	}

	outDir := flag.Args()[2]

	if err := os.RemoveAll(outDir); err != nil {
		log.Fatalf("RemoveAll: %v", err)
	}
	if err := os.MkdirAll(outDir, 0777); err != nil {
		log.Fatalf("MkdirAll: %v", err)
	}

	result, err := compareDir(flag.Args()[0],
		flag.Args()[1], *fileRegexp)
	if err != nil {
		log.Fatalf("compareDir: %v", err)
	}
	if err := result.renderPNG(outDir); err != nil {
		log.Fatal("renderPNG: ", err)
	}
	if err := result.comparePNG(outDir, *cmpJobs); err != nil {
		log.Fatal("comparePNG: ", err)
	}

	result.Trim(*maxOutputs)
	result.LinkFiles(outDir)
	result.DumpTXT(os.Stdout)
	if err := result.DumpHTMLFile(outDir); err != nil {
		log.Fatal("DumpHTMLFile", err)
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

func (fr *fileResult) compareOneIM() error {
	cmd := exec.Command("compare", "-verbose", "-metric", "MAE",
		fr.In[0], fr.In[1], fr.out)

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
			return err
		}
	}

	str := stdout.String()

	submatch := imageMagickRE.FindStringSubmatch(str)
	if len(submatch) != 2 {
		return fmt.Errorf("missing re")
	}

	dist, err := strconv.ParseFloat(submatch[1], 64)
	fr.Dist = dist
	return err
}

func (fr *fileResult) compareOne() error {
	if *imageMagick {
		return fr.compareOneIM()
	}

	f1, err := os.Open(fr.In[0])
	if err != nil {
		return err
	}
	defer f1.Close()

	i1, err := png.Decode(f1)
	if err != nil {
		return err
	}
	f2, err := os.Open(fr.In[1])
	if err != nil {
		return err
	}
	defer f2.Close()
	i2, err := png.Decode(f2)
	if err != nil {
		return err
	}
	diff, dist, err := ImageCompareConvolve(asRGBA(i1), asRGBA(i2), filepath.Base(fr.In[0]))
	if err != nil {
		return err
	}
	_, mae, err := ImageCompareMAE(asRGBA(i1), asRGBA(i2))
	if err != nil {
		return err
	}

	fr.Dist = dist
	fr.DistMAE = mae
	if dist > 0 {
		os.MkdirAll(filepath.Dir(fr.out), 0755)

		outF, err := os.OpenFile(fr.out, os.O_WRONLY|os.O_CREATE, 0666)
		if err != nil {
			return err
		}
		if err := png.Encode(outF, diff); err != nil {
			return err
		}
		return outF.Close()
	}
	return nil
}
