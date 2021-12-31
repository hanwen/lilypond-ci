package main

import (
	"bytes"
	"flag"
	"fmt"
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

var startBranch string

func getCommit(version string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--short", version)
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

func checkout(version string) error {
	cmd := exec.Command("git", "checkout", version)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func describe(v string) string {
	cmd := exec.Command("git", "log", "-1", "--pretty=format:%s", v)
	out, err := cmd.Output()
	check(err)
	return strings.TrimSpace(string(out))
}

func workdirDiff() (string, error) {
	cmd := exec.Command("git", "diff")
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

func getBranch() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--symbolic-full-name", "HEAD")
	out, err := cmd.Output()
	branch := strings.TrimSpace(string(out))
	branch = strings.TrimPrefix(branch, "refs/heads/")
	return branch, err
}

func getCPUType() (string, error) {
	content, err := ioutil.ReadFile("/proc/cpuinfo")
	if err != nil {
		return "", err
	}

	for _, l := range strings.Split(string(content), "\n") {
		fields := strings.SplitN(l, "\t: ", 2)
		if fields[0] == "model name" {
			return fields[1], nil
		}
	}

	return "unknown", nil
}

func getCPUSpeedsKhz() (speeds map[string]int, err error) {
	keys := []string{"scaling_max_freq",
		"scaling_min_freq",
		"cpuinfo_min_freq",
		"cpuinfo_max_freq",
	}

	speeds = make(map[string]int)
	for _, k := range keys {
		content, err := ioutil.ReadFile("/sys/devices/system/cpu/cpu0/cpufreq/" + k)
		if err != nil {
			return nil, err
		}
		spd, err := strconv.Atoi(strings.TrimSpace(string(content)))
		if err != nil {
			return nil, err
		}
		speeds[k] = spd
	}
	return speeds, nil
}

func setCPUSpeed(minKhz, maxKhz int) error {
	min := fmt.Sprintf("%dkhz", minKhz)
	max := fmt.Sprintf("%dkhz", maxKhz)
	cmd := exec.Command("sudo", "-n", "cpupower", "frequency-set", "-u", max, "-d", min)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	log.Printf("running %s", cmd.Args)
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func Error(msg string) {
	fmt.Println(msg)
	if startBranch != "" {
		fmt.Println("back to branch", startBranch)
		checkout(startBranch)
	}
	os.Exit(1)
}

func check(err error) {
	if err != nil {
		Error(err.Error())
	}
}

func sanityCheck() {
	diff, err := workdirDiff()
	check(err)
	if diff != "" {
		Error("should be clean")
	}
	content, err := ioutil.ReadFile("config.make")
	check(err)
	if !bytes.Contains(content, []byte("-O2")) {
		Error("need -O2 for benchmarking")
	}
}

func Shell(shcmd string) error {
	cmd := exec.Command("/bin/sh", "-c", shcmd)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func buildVersion(v string) error {
	if err := checkout(v); err != nil {
		return err
	}
	if content, err := ioutil.ReadFile("config.make"); err != nil {
		return err
	} else if !strings.Contains(string(content), "-O2") {
		return fmt.Errorf("config.mark is missing -O2")
	}
	bin := fmt.Sprintf("binaries/lilypond.%s", v)
	callBin := fmt.Sprintf("out/bin/lilypond.%s", v)
	if _, err := os.Lstat(bin); err != nil {
		if err := Shell(fmt.Sprintf("make -C flower/ clean && make -C lily/ clean && make -C flower -j4 && make -C lily/ -j4 && cp lily/out/lilypond %s", bin)); err != nil {
			return err
		}
	}
	os.Remove(callBin)
	if err := os.Link(bin, callBin); err != nil {
		return err
	}

	return nil
}

type runResults struct {
	mem  float64
	time float64
}

type allResults struct {
	baseMem  []float64
	versMem  []float64
	baseTime []float64
	versTime []float64
	runCount int
	args     []string
}

var (
	timeRe = regexp.MustCompile(`User time \(seconds\): ([0-9.]+)`)
	memRe  = regexp.MustCompile(`Maximum resident set size.*: ([0-9.]+)`)
)

func commandId(args []string) string {
	return regexp.MustCompile("[^a-zA-Z0-9-_]").ReplaceAllString(time.Now().Format(time.RFC3339)+strings.Join(args, ""), "")
}

func benchmark(v1, v2 string, count int, outDir string, args []string) allResults {
	mem := map[string][]float64{}
	time := map[string][]float64{}

	for i := 0; i < count; i++ {
		for _, v := range []string{v1, v2} {
			out := fmt.Sprintf("%s/%s-%s.%d.txt", outDir, commandId(args), v, i)

			check(checkout(v))
			cmd := fmt.Sprintf("/usr/bin/time -v out/bin/lilypond.%s %s >& %s", v, strings.Join(args, " "), out)
			log.Println("running", cmd)
			check(Shell(cmd))

			content, err := ioutil.ReadFile(out)
			check(err)
			cstr := string(content)
			match := timeRe.FindStringSubmatch(cstr)
			t, err := strconv.ParseFloat(match[1], 64)
			check(err)
			match = memRe.FindStringSubmatch(cstr)
			m, err := strconv.ParseFloat(match[1], 64)
			check(err)

			mem[v] = append(mem[v], m)
			time[v] = append(time[v], t)
		}
	}

	return allResults{
		baseMem:  mem[v1],
		baseTime: time[v1],
		versMem:  mem[v2],
		versTime: time[v2],
		args:     args,
		runCount: count,
	}
}

func stddev(fs []float64) float64 {
	a := mean(fs)
	tot := 0.0
	for _, f := range fs {
		tot += (f - a) * (f - a)
	}
	return math.Sqrt(tot / float64(len(fs)-1))
}

func mean(fs []float64) float64 {
	tot := 0.0
	for _, f := range fs {
		tot += f
	}
	return tot / float64(len(fs))
}

func analyzeData(base []float64, data []float64, name string, version string) string {
	sort.Float64s(base)
	sort.Float64s(data)

	baseMed := base[len(base)/2]
	versMed := data[len(data)/2]

	delta := versMed - baseMed
	baseStddev := stddev(base)
	dataStddev := stddev(data)

	qual := "neutral"
	if math.Abs(delta) > baseStddev+dataStddev {
		if name == "mem" {
			if delta > 0 {
				qual = "fatter"
			} else {
				qual = "leaner"
			}
		}
		if name == "time" {
			if delta > 0 {
				qual = "slower"
			} else {
				qual = "faster"
			}
		}
	}

	return fmt.Sprintf(`    %s delta: %f (stddev %f %f n=%d)
    %s delta: %f %% (%s is %s)`, name, delta, baseStddev, dataStddev, len(base), name, 100.0*delta/baseMed, version, qual)
}

func analyze(base, vers string, res allResults) string {
	speeds, _ := getCPUSpeedsKhz()
	cpu, _ := getCPUType()
	if strings.Contains(cpu, "@") {
		fs := strings.SplitN(cpu, "@", 2)
		cpu = fs[0]
	}

	r := fmt.Sprintf(`benchmark for arguments: %s

%s at %d Mhz

raw data (%s):
   %v %v
raw data (%s):
   %v %v
`, res.args,
		cpu, speeds["scaling_min_freq"]/1000,
		base, res.baseMem, res.baseTime,
		vers, res.versMem, res.versTime)

	r += fmt.Sprintf(`%s - %s
    baseline %s - %s
    args %s
%s
%s
`, vers, describe(vers), base, describe(base), res.args,
		analyzeData(res.baseMem, res.versMem, "mem", vers),
		analyzeData(res.baseTime, res.versTime, "time", vers))

	return r
}

func main() {
	version := flag.String("version", "HEAD", "version to test")
	baseline := flag.String("baseline", "", "baseline")
	runCount := flag.Int("run_count", 3, "run count")
	outDir := flag.String("out", "benchmark-results", "out dir")

	flag.Parse()
	speeds, err := getCPUSpeedsKhz()
	check(err)

	if min, max := speeds["scaling_min_freq"], speeds["scaling_max_freq"]; min != max {
		if err := setCPUSpeed(speeds["cpuinfo_max_freq"]/2, speeds["cpuinfo_max_freq"]/2); err != nil {
			log.Fatalf("setCPUSpeed %v", err)
		}
		defer setCPUSpeed(speeds["cpuinfo_min_freq"], speeds["cpuinfo_max_freq"])
	} else {
		log.Printf("CPU freq %d Mhz", min/1000)
	}
	if *baseline == "" {
		*baseline, err = getCommit(*version + "^")
		check(err)
	}
	*baseline, err = getCommit(*baseline)
	check(err)
	*version, err = getCommit(*version)
	check(err)
	startBranch, err = getBranch()
	check(err)
	log.Println("startbranch is", startBranch)
	sanityCheck()

	for _, v := range []string{*baseline, *version} {
		check(buildVersion(v))
	}

	result := benchmark(*baseline, *version, *runCount, *outDir, flag.Args())
	log.Println(result)
	summary := analyze(*baseline, *version, result)
	log.Println(summary)
	check(ioutil.WriteFile(filepath.Join(*outDir, fmt.Sprintf("%s-v%s-base%s", commandId(flag.Args()),
		*version, *baseline)), []byte(summary), 0644))
	if startBranch != "" {
		checkout(startBranch)
	}
}
