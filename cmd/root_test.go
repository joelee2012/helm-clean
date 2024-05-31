package cmd

import (
	"bytes"
	"os"
	"os/exec"
	"path"
	"slices"
	"strings"
	"testing"
	"time"
)

func mockHelm(t *testing.T) {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		t.Errorf("mock helm failed: %s", err)
	}
	os.Setenv("HELM_BIN", path.Join(string(bytes.Trim(out, "\n")), "testdata/helm"))
}

func TestListRelease(t *testing.T) {
	duration, _ := time.ParseDuration("240h")
	c := Clean{Before: duration, DryRun: true}
	mockHelm(t)
	rList, err := c.ListRelease()
	if err != nil {
		t.Errorf("list release failed with error: %s", err)
	}
	var names []string
	for _, r := range rList {
		names = append(names, r.Name)
	}
	if !slices.Contains(names, "release-c") {
		t.Errorf("expect release-c is in release list %s", names)
	}
}

func TestRun(t *testing.T) {
	mockHelm(t)
	duration, _ := time.ParseDuration("240h")
	c := Clean{Before: duration, DryRun: true}
	var w bytes.Buffer
	c.Run(&w)
	if !strings.Contains(w.String(), "ns-1,release-c") {
		t.Errorf("expect: 'ns-1,release-c' in output, but got: %s", w.String())
	}

	c.DryRun = false
	w.Reset()
	c.Run(&w)
	if w.String() != "uninstall -n ns-1 release-c\n" {
		t.Errorf("expect: uninstall -n ns-1 release-c, but got: %s", w.String())
	}
}

func executeCmd(args []string) (string, string, error) {
	cmd := newRootCmd()
	var o, e bytes.Buffer
	cmd.SetOut(&o)
	cmd.SetErr(&e)
	cmd.SetArgs(args)
	err := cmd.Execute()
	if err != nil {
		return "", "", err
	}
	return o.String(), e.String(), err
}

func TestNewRootCmd(t *testing.T) {
	o, _, err := executeCmd([]string{"-h"})
	if err != nil {
		t.Errorf("execute rootcmd failed: %s", err)
	}
	if !strings.Contains(o, "A helm plugin to clean release by date") {
		t.Errorf("expect x, but got: %s", o)
	}
	_, _, err = executeCmd([]string{"-b", "1"})
	if !strings.Contains(err.Error(), "missing unit in duration") {
		t.Errorf("expect missing unit in duration, but got: %s", err)
	}
}
