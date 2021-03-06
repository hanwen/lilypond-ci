package main

import (
	"archive/zip"
	"encoding/json"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	gitlab "github.com/xanzy/go-gitlab"
)

type metadata struct {
	JobFinishedAt time.Time
	UnpackedAt    time.Time
	JobID         int
}

type loggingRoundTripper struct {
	http.RoundTripper
}

func (c *loggingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	log.Printf("HTTP %s %s", req.Method, req.URL)
	return c.RoundTripper.RoundTrip(req)
}

func loggingHTTP() *http.Client {
	loggingClient := *http.DefaultClient
	loggingClient.Transport = &loggingRoundTripper{
		http.DefaultTransport,
	}
	return &loggingClient
}

func main() {
	tokenFile := flag.String("token", "", "file with token")
	destDir := flag.String("dir", "", "destination dir")
	stripCount := flag.Int("strip", 3, "number of leading dir components to strip")
	repo := flag.String("repo", "hanwenn/lilypond", "repository to fetch artifact for")
	flag.Parse()

	if *tokenFile == "" {
		log.Fatal("must specify --token")
	}
	if *destDir == "" {
		log.Fatal("must specify --dir")
	}
	*destDir = filepath.Clean(*destDir)

	tokenBytes, err := ioutil.ReadFile(*tokenFile)
	if err != nil {
		log.Fatal(err)
	}

	token := strings.TrimSpace(string(tokenBytes))

	client, err := gitlab.NewClient(token, gitlab.WithHTTPClient(loggingHTTP()))
	if err != nil {
		log.Fatal(err)
	}

	metadataFile := filepath.Join(*destDir, "artifact.json")
	var lastJobID int
	if c, err := ioutil.ReadFile(metadataFile); err == nil {
		var m metadata
		if err := json.Unmarshal(c, &m); err == nil {
			// Add some padding so we don't find the last run.
			lastJobID = m.JobID
		}
	}

	jobs, rep, err := client.Jobs.ListProjectJobs(*repo,
		&gitlab.ListJobsOptions{
			Scope: []gitlab.BuildStateValue{gitlab.Success},
		})
	if err != nil {
		log.Fatalf("ListProjectJobs: %v", err)
	}

	if rep.TotalItems == 0 {
		log.Printf("no items found; bailing.")
		os.Exit(0)
	}

	var metadata metadata
found:
	for _, j := range jobs {
		if j.Stage != "website" {
			continue
		}
		if j.ID <= lastJobID {
			// The API docs suggest that IDs are ever-increasing
			log.Printf("no newer jobs found. bailing")
			os.Exit(0)
		}
		log.Printf("J %#v", j)
		for _, a := range j.Artifacts {
			if a.Filename == "website.zip" {
				metadata.JobID = j.ID
				metadata.JobFinishedAt = *j.FinishedAt
				break found
			}
		}
	}

	archive, err := download(client, *destDir, *repo, metadata.JobID)
	if err != nil {
		log.Fatal(err)
	}

	err = unpack(archive, *destDir, *stripCount, metadata)
	os.Remove(archive)
	if err != nil {
		log.Fatal(err)
	}
}

func download(client *gitlab.Client, destDir string, repoID string, jobID int) (string, error) {
	f, err := ioutil.TempFile(filepath.Dir(destDir), "gitlab-artifact-zip")
	if err != nil {
		return "", err
	}

	rm := f.Name()
	defer func() {
		if rm != "" {
			os.Remove(rm)
		}
	}()

	r, _, err := client.Jobs.GetJobArtifacts(repoID, jobID)
	if err != nil {
		return "", err
	}

	if _, err := io.Copy(f, r); err != nil {
		return "", err
	}

	rm = ""
	return f.Name(), nil
}

func unpack(archive, destDir string, stripCount int, metadata metadata) error {
	tmp, err := ioutil.TempDir(filepath.Dir(destDir), "gitlab-artifact-dir")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	zr, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}
	defer zr.Close()

	for _, zf := range zr.File {
		if zf.FileInfo().IsDir() {
			continue
		}

		dst := zf.Name
		components := strings.Split(dst, "/")
		if len(components) <= stripCount {
			continue
		}
		components = components[stripCount:]
		dst = filepath.Join(tmp, strings.Join(components, "/"))
		if err := os.MkdirAll(filepath.Dir(dst), 0777); err != nil {
			return err
		}

		f, err := os.Create(dst)
		if err != nil {
			return err
		}
		zc, err := zf.Open()
		if err != nil {
			return err
		}
		if _, err := io.Copy(f, zc); err != nil {
			return err
		}
		zc.Close()
		if err := f.Close(); err != nil {
			return err
		}
	}

	metadata.UnpackedAt = time.Now()
	if c, err := json.Marshal(&metadata); err != nil {
		return err
	} else if err := ioutil.WriteFile(filepath.Join(tmp, "artifact.json"), c, 0666); err != nil {
		return err
	}

	removeMe := ""
	if _, err := os.Lstat(destDir); err == nil {
		removeMe = destDir + ".old"
		if err := os.Rename(destDir, removeMe); err != nil {
			return err
		}
	}
	if err := os.Rename(tmp, destDir); err != nil {
		return err
	}

	if removeMe != "" {
		os.RemoveAll(removeMe)
	}

	return nil
}
