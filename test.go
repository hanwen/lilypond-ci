package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var (
	allPlatforms = []string{"ubuntu", "fedora", "fedora-guile2"}
	allModes     = []string{"incremental", "full", "separate"}
	allStages    = []string{"build", "check", "doc"}
	dockerFiles  = map[string]string{
		"ubuntu":        "ubuntu-xenial.dockerfile",
		"fedora":        "fedora-31.dockerfile",
		"fedora-guile2": "fedora-31-guile2.dockerfile",
	}
)

type platformSetting struct {
	Tag        string
	Dockerfile string
}

func known(ss []string, s string) bool {
	for _, c := range ss {
		if c == s {
			return true
		}
	}
	return false
}

type rietveldData struct {
	Patchsets []int `json:"patchsets"`
}

func system(cmd string) error {
	c := exec.Command("/bin/bash", "-euc", cmd)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	log.Printf("command %v", c.Args)
	return c.Run()
}

func patchRietveldChange(changeNum int) (branch string, err error) {
	resp, err := http.Get(fmt.Sprintf("https://codereview.appspot.com/api/%d/", changeNum))
	if err != nil {
		return "", err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	resp.Body.Close()
	var rv rietveldData
	if err := json.Unmarshal(body, &rv); err != nil {
		return "", err
	}
	if len(rv.Patchsets) == 0 {
		return "", errors.New("no patchsets")
	}
	patchset := rv.Patchsets[len(rv.Patchsets)-1]
	issue := fmt.Sprintf("issue%d_%d", changeNum, patchset)
	if err := system(fmt.Sprintf(`ISSUE=%s
		cd lilypond
		git fetch origin
		git checkout -f origin/master
		(git branch -D $ISSUE || true)
		git checkout -b $ISSUE origin/master`, issue)); err != nil {
		return "", err
	}
	if err := system(fmt.Sprintf(`ISSUE=%s
cd lilypond && curl https://codereview.appspot.com/download/${ISSUE}.diff | git apply
git add .
git commit -m "${ISSUE}" -a`, issue)); err != nil {
		system("cd lilypond; git am --abort")
		return "", err
	}
	return issue, nil
}

func testOne(platform, mode, stage, url, branch string, timeout time.Duration) (dest string, err error) {
	driverScript := fmt.Sprintf("test-%s.sh", mode)
	seedImage := "lilypond-base-" + platform
	if mode == "incremental" {
		seedImage = "lilypond-seed-" + platform
	}

	localRepo := "/local"
	log.Println("***")
	log.Printf("Testing %s %s for %s mode %s stage %s", url, branch, platform, mode, stage)
	log.Println("***")

	containerURL := "/local"
	if fi, err := os.Stat(url); err == nil && fi.IsDir() && url != "lilypond" {
		url, err = filepath.Abs(url)
		if err != nil {
			return "", err
		}
		if err := system(fmt.Sprintf(`cd lilypond &&
git checkout -f $(git rev-parse origin/master) &&
git fetch -f %s %s:%s`, url, branch, branch)); err != nil {
			return "", err
		}
	}

	name := regexp.MustCompile("^.*:").ReplaceAllString(url+"_"+branch, "")
	name = regexp.MustCompile("[:/ ]").ReplaceAllString(name, "-")
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dest = filepath.Join(cwd, "../lilypond-test-results", name, seedImage)
	if err := os.MkdirAll(dest, 0777); err != nil {
		return "", err
	}

	if timeout == 0 {
		timeout = 24 * 60 * 60 * time.Second
	}
	cmd := exec.Command("docker",
		"run", "-v", dest+":/output", "-v", filepath.Join(cwd, "lilypond")+":/"+localRepo+":ro",
		"-v", filepath.Join(cwd, driverScript)+":/test.sh:ro", "--rm=true",
		seedImage, "timeout", "--signal=KILL", fmt.Sprintf("%f", timeout.Seconds()), "/test.sh", stage, containerURL, branch, localRepo, "origin/master")
	r, w, err := os.Pipe()
	if err != nil {
		return "", err
	}
	cmd.Stdout = w
	cmd.Stderr = w
	defer w.Close()
	defer r.Close()

	// closing?
	logFilename := filepath.Join(dest, "log.txt")
	log.Printf("logfile in %s", logFilename)
	logFile, err := os.Create(logFilename)
	if err != nil {
		return "", err
	}
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := r.Read(buf)
			_, err2 := logFile.Write(buf[:n])
			os.Stdout.Write(buf[:n])
			if err2 != nil {
				log.Printf("log write: %v", err)
				break
			}
			if err != nil {
				break
			}
		}
		logFile.Close()
	}()

	log.Printf("running: %v", cmd.Args)
	if err := cmd.Run(); err != nil {
		return "", err
	}

	log.Printf("results in %s", dest)

	return dest, nil
}

func main() {
	platform := flag.String("platform", "ubuntu", "platform to test on: "+strings.Join(allPlatforms, " "))
	mode := flag.String("mode", "incremental", "how to build: "+strings.Join(allModes, " "))
	stage := flag.String("stage", "check", "which stage to execute: "+strings.Join(allStages, " "))
	doTest := flag.Bool("test", true, "test a change")
	doRebase := flag.Bool("rebase", false, "recreate base image")
	doReseed := flag.Bool("reseed", false, "recreate seed image")
	rietveld := flag.Int("rietveld", 0, "rietveld change number")
	timeout := flag.Duration("timeout", 0, "timeout for the subprocess")
	flag.Parse()

	var platforms []string
	for _, p := range strings.Split(*platform, ",") {
		if p == "all" {
			platforms = allPlatforms
			break
		}
		if p == "guile2" {
			p = "fedora-guile2"
		}
		if !known(allPlatforms, p) {
			log.Fatalf("unknown platform %q", *platform)
		}
		platforms = append(platforms, p)
	}

	if *doReseed {
		// todo - check if base image exists.
		for _, p := range platforms {
			if err := system(fmt.Sprintf(`
		(cd lilypond && git fetch)
		docker tag lilypond-base-%s lilypond-base
		docker build -t lilypond-seed-%s -f lilypond-seed.dockerfile .
`, p, p)); err != nil {
				log.Fatalf("system: %v", err)
			}
		}
	} else if *doRebase {
		for _, p := range platforms {
			if err := system(fmt.Sprintf("docker build --no-cache -t lilypond-base-%s -f %s .", p, dockerFiles[p])); err != nil {
				log.Fatalf("system: %v", err)
			}
		}
	} else if *doTest {
		if !known(allModes, *mode) {
			log.Fatalf("unknown mode %q", *mode)
		}
		if !known(allStages, *stage) {
			log.Fatalf("unknown stage %q", *stage)
		}

		if *rietveld == 0 && len(flag.Args()) != 2 {
			log.Fatal("Need URL BRANCH or 'rietveld' CHANGE-NUM")
		}
		repoURL := flag.Arg(0)
		branch := flag.Arg(1)
		if *rietveld != 0 {
			var err error
			branch, err = patchRietveldChange(*rietveld)
			if err != nil {
				log.Fatalf("patchRietveldChange: %v", err)
			}
			repoURL = "lilypond"
		}
		for _, p := range platforms {
			_, err := testOne(p, *mode, *stage, repoURL, branch, *timeout)
			if err != nil {
				log.Fatalf("testOne (%s): %v", p, err)
			}
		}
	} else {
		log.Fatal("must specify --test, --rebase or --reseed")
	}
}
