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
	"io/ioutil"
	"log"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

func sqDiffUInt8(x, y uint8) uint64 {
	d := int64(x) - int64(y)
	return uint64(d * d)
}

func sqDiffRGBA(p, q color.RGBA) uint64 {
	r := int64(p.R) - int64(q.R)
	g := int64(p.G) - int64(q.G)
	b := int64(p.B) - int64(q.B)
	a := int64(p.A) - int64(q.A)
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

var digitRE = regexp.MustCompile("-[0-9][0-9]*.(eps|png)$")

func readDir(dir, ext string) (map[string]struct{}, error) {
	d, err := filepath.Glob(filepath.Join(flag.Args()[0], "*."+ext))
	if err != nil {
		return nil, err
	}
	result := make(map[string]struct{}, len(d))
	for _, n := range d {
		b := filepath.Base(n)
		if digitRE.FindString(b) == "" {
			continue
		}
		result[b] = struct{}{}
	}

	return result, nil
}

var batchGS = flag.Bool("batch_gs", true, "")
var localDataDir = flag.Bool("local", false, "")
var verbose = flag.Bool("verbose", false, "")

func convertEPSParallel(eps_files []string, ncpu int) (map[string]string, error) {
	sz := len(eps_files) / ncpu
	if sz == 0 {
		sz++
	}

	chunks := make([][]string, ncpu)
	for i, f := range eps_files {
		chunks[i%ncpu] = append(chunks[i%ncpu], f)
	}

	type chunkResult struct {
		filemap map[string]string
		err     error
	}
	done := make(chan chunkResult, ncpu)
	for _, chunk := range chunks {
		go func(ch []string) {
			var r chunkResult
			r.filemap, r.err = convertEPS(ch)
			done <- r
		}(chunk)
	}

	result := map[string]string{}
	for range chunks {
		r := <-done
		if r.err != nil {
			return nil, r.err
		}

		for k, v := range r.filemap {
			result[k] = v
		}
	}
	return result, nil
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

func convertEPS(epsFiles []string) (map[string]string, error) {
	if *batchGS {
		return convertEPSBatch(epsFiles)
	}

	result := map[string]string{}
	for _, f := range epsFiles {
		r, err := convertEPSBatch([]string{f})
		if err != nil {
			return nil, err
		}
		result[f] = r[f]
	}

	return result, nil
}

func convertEPSBatch(epsFiles []string) (map[string]string, error) {
	dataOption := ""
	if *localDataDir {
		doneDir := map[string]bool{}
		for _, fn := range epsFiles {
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
				return nil, err
			}
			dir, err = filepath.Abs(dir)
			if err != nil {
				return nil, err
			}
			if fi.IsDir() {
				dataOption = fmt.Sprintf("-slilypond-datadir=%s/share/lilypond/current", dir)
				break
			}
		}
	}

	emptyPS, err := ioutil.TempFile("", "emptyps")
	if err != nil {
		return nil, err
	}
	if err := ioutil.WriteFile(emptyPS.Name(), []byte(`%!PS-Adobe-3.0 EPSF-3.0
%%BoundingBox: 0 0 1 1
%%EndComments
`), 0644); err != nil {
		return nil, err
	}
	emptyPS.Close()
	driver, err := ioutil.TempFile("", "emptyps")
	if err != nil {
		return nil, err
	}

	result := make(map[string]string, len(epsFiles))
	for _, fn := range epsFiles {
		inputFn := fn
		if empty, err := EPSBBoxEmpty(fn); err != nil {
			return nil, err
		} else if empty {
			inputFn = emptyPS.Name()
		}

		outFn := strings.Replace(fn, ".eps", ".png", 1)
		verbosePS := ""
		if *verbose {
			verbosePS = fmt.Sprintf(" (processing %s\n) print ", fn)
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
			return nil, err
		}
		result[fn] = outFn
	}

	driver.Close()
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
		return nil, err
	}
	return result, nil
}

func compareDirEPS(in1, in2, out string) (*compareResult, error) {
	start := time.Now()
	epsFileCount := 0
	for _, dir := range []string{in1, in2} {
		epsFiles, err := readDir(dir, "eps")
		if err != nil {
			return nil, err
		}

		var keys []string
		for k := range epsFiles {
			keys = append(keys, filepath.Join(dir, k))
		}
		sort.Strings(keys)
		epsFileCount += len(keys)
		if _, err := convertEPSParallel(keys, *gsJobs); err != nil {
			return nil, err
		}
	}
	epsDT := time.Now().Sub(start)
	log.Printf("Convert %d EPS files using %d cores (batch=%v) to PNG in %v (%v/file)", epsFileCount, *gsJobs, *batchGS, epsDT, epsDT/(1+time.Duration(epsFileCount)))

	return compareDirPNG(in1, in2, out, *cmpJobs)
}

type compareResult struct {
	Results []fileResult
}

func (r *compareResult) Trim(max int) {
	sort.Slice(r.Results, func(i, j int) bool { return r.Results[i].Dist > r.Results[j].Dist })
	for i := range r.Results {
		if r.Results[i].Dist == 0.0 || i > max {
			r.Results = r.Results[:i]
			break
		}
	}
}

func (r *compareResult) LinkFiles(outDir string) error {
	for _, r := range r.Results {
		if err := os.Link(r.in1, filepath.Join(outDir, r.Name+".1.png")); err != nil {
			return err
		}
		if err := os.Link(r.in2, filepath.Join(outDir, r.Name+".2.png")); err != nil {
			return err
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
	Name     string
	Dist     float64
	in1, in2 string
	out      string
	err      error
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
      <tr><th>old</th><th>new</th></tr>
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
    <img src="{{.Name}}.1.png">
    <br>
    {{.Name}}
  </td>
  <td>
    <div>
      <div style="position: absolute">
         <img src="{{.Name}}.2.png">
      </div>
      <div style="position: absolute; opacity: 0.3">
         <img src="{{.Name}}.diff.png">
      </div>
      <div style="opacity: 0.0">
         <img src="{{.Name}}.diff.png">
      </div>
    </div>
    <br>
    {{.Name}}
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

func compareDirPNG(in1, in2, outDir string, ncpu int) (*compareResult, error) {
	start := time.Now()
	dir1, err := readDir(in1, "png")
	if err != nil {
		return nil, err
	}
	dir2, err := readDir(in2, "png")
	if err != nil {
		return nil, err
	}

	var names []string
	for k := range dir1 {
		if _, ok := dir2[k]; ok {
			names = append(names, k)
		}
	}

	sort.Strings(names)
	todo := make(chan *fileResult, len(names))
	done := make(chan *fileResult, len(names))
	for _, k := range names {
		nm := k[:len(k)-len(filepath.Ext(k))]
		todo <- &fileResult{
			Name: nm,
			in1:  filepath.Join(in1, k),
			in2:  filepath.Join(in2, k),
			out:  filepath.Join(outDir, nm+".diff.png"),
		}
	}
	close(todo)

	for i := 0; i < ncpu; i++ {
		go func() {
			for t := range todo {
				t.Dist, t.err = compareOne(t.in1, t.in2, t.out)
				done <- t
			}
		}()
	}

	var result compareResult
	for range names {
		result.Results = append(result.Results, *<-done)
	}
	pngDT := time.Now().Sub(start)
	log.Printf("compared %d PNG image pairs using %d cores (imagemagick=%v) in %v (%v / pair)", len(names), ncpu, *imageMagick, pngDT, pngDT/time.Duration(len(names)))

	for _, r := range result.Results {
		if r.err != nil {
			return nil, r.err
		}
	}
	return &result, nil
}

var (
	gsJobs      = flag.Int("gs_jobs", 1, "")
	cmpJobs     = flag.Int("cmp_jobs", 1, "")
	imageMagick = flag.Bool("imagemagick", false, "")
	max         = flag.Int("max", 0, "output top-N differences")
)

func main() {
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
	result, err := compareDirEPS(flag.Args()[0],
		flag.Args()[1],
		outDir)
	if err != nil {
		log.Fatal("compareDir: ", err)
	}

	result.Trim(*max)
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
