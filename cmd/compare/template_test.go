package main

import (
	"bytes"
	"testing"
)

func TestTemplate(t *testing.T) {
	r := compareResult{
		Results: []*fileResult{
			{
				Name: "myname",
				Dist: 1.2,
			},
		},
	}
	buf := bytes.Buffer{}
	if err := r.DumpHTML(&buf); err != nil {
		t.Fatalf("DumpHTML: %v", err)
	}
}
