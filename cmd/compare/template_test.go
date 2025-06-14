package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestTemplate(t *testing.T) {
	r := compareResult{
		Results: []*fileResult{
			&fileResult{
				Name: "myname",
				Dist: 1.2,
			},
		},
	}
	buf := bytes.Buffer{}
	if err := r.DumpHTML(&buf); err != nil {
		t.Fatalf("DumpHTML: %v", err)
	}

	want := `
<html>
  <style>
    table, th, td {
      border: 1px solid grey;
    }
  </style>
  <title>Image comparison</title>
  <body>
    <table>
      <th><td>old</td><td>new</td></th>
      
         
<tr>
  <td>
    <img src="myname.1.png">
    <br>
    myname
  </td>
  <td>
    <div>
      <div style="position: absolute">
         <img src="myname.2.png">
      </div>
      <div style="position: absolute; opacity: 0.3">
         <img src="myname.diff.png">
      </div>
      <div style="opacity: 0.0">
         <img src="myname.diff.png">
      </div>
    </div>
  </td>
</tr>
        
      
    </table>
  </body>
</html>
`
	got := buf.String()
	if got != want {
		l1 := strings.Split(got, "\n")
		l2 := strings.Split(want, "\n")
		for i := 0; i < len(l1) && i < len(l2); i++ {
			if l1[i] != l2[i] {
				t.Errorf("line %d: got %q != want %q", i, l1[i], l2[i])
			}
		}
		if len(l1) != len(l2) {
			t.Errorf("len %d != %d", len(l1), len(l2))
		}
	}
}
