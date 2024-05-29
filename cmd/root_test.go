package cmd

import (
	"bytes"
	"os"
	"os/exec"
	"path"
	"slices"
	"testing"
)

func TestListRelease(t *testing.T) {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		t.Errorf("get root folder failed: %s", err)
	}
	os.Setenv("HELM_BIN", path.Join(string(bytes.Trim(out, "\n")), "testdata/helm.sh"))
	c := Clean{Before: "10d", DryRun: true}
	rList, err := c.ListRelease()
	if err != nil {
		t.Errorf("list release failed with error: %s", err)
	}
	var names []string
	for _, r := range rList {
		names = append(names, r.Name)
	}
	if !slices.Contains(names, "tboxsimulator-home") {
		t.Errorf("appserver-home is not in list")
	}
}
