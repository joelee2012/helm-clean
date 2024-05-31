package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"

	"github.com/spf13/cobra"
)

func newRootCmd() *cobra.Command {
	var clean = Clean{}
	var rootCmd = &cobra.Command{
		Use:     "clean",
		Short:   "A helm plugin to clean release by date",
		Long:    `A helm plugin to clean release by date`,
		Version: "0.0.1",
		RunE: func(cmd *cobra.Command, args []string) error {
			return clean.Run(os.Stdout)
		},
	}
	rootCmd.Flags().DurationVarP(&clean.Before, "before", "b", 0, "The last updated time before now, eg: 8h")
	rootCmd.Flags().StringVarP(&clean.Filter, "filter", "f", "", "A regular expression, The chart of releases that match the expression will be included in the results")
	rootCmd.Flags().BoolVarP(&clean.DryRun, "dry-run", "d", true, "Dry run mode only print the release info")
	rootCmd.Flags().BoolVarP(&clean.AllNamespace, "all-namespaces", "A", false, "List releases across all namespaces")
	return rootCmd
}

func Execute() {
	err := newRootCmd().Execute()
	if err != nil {
		os.Exit(1)
	}
}

type Clean struct {
	Before       time.Duration
	DryRun       bool
	Filter       string
	AllNamespace bool
}

type Release struct {
	AppVersion                                        string `json:"app_version"`
	Chart, Name, Namespace, Status, Updated, Revision string
}

type ReleaseList []*Release

var timeFormat = "2006-01-02 15:04:05 UTC"

func (c *Clean) ListRelease() (ReleaseList, error) {
	args := []string{"list", "--no-headers", "-o", "json", "--time-format", timeFormat}
	if c.AllNamespace {
		args = append(args, "-A")
	}
	out, err := exec.Command(os.Getenv("HELM_BIN"), args...).Output()
	if err != nil {
		return nil, err
	}
	var rList ReleaseList
	if err := json.Unmarshal(out, &rList); err != nil {
		return nil, err
	}
	now := time.Now()
	var result ReleaseList
	pattern := regexp.MustCompile(c.Filter)
	loc, err := time.LoadLocation("Local")
	if err != nil {
		return nil, err
	}
	for _, release := range rList {
		t, err := time.ParseInLocation(timeFormat, release.Updated, loc)
		if err != nil {
			return nil, err
		}
		if now.After(t.Add(c.Before)) && pattern.MatchString(release.Chart) {
			result = append(result, release)
		}
	}
	return result, nil
}

func (c *Clean) Run(w io.Writer) error {
	rList, err := c.ListRelease()
	if err != nil {
		return err
	}
	if c.DryRun {
		t := table.NewWriter()
		t.SetOutputMirror(w)
		t.AppendHeader(table.Row{"NAMESPACE", "NAME", "UPDATED", "CHART", "APP VERSION"})
		for _, release := range rList {
			t.AppendRow(table.Row{release.Namespace, release.Name, release.Updated, release.Chart, release.AppVersion})
		}
		t.RenderCSV()

	} else {
		for _, release := range rList {
			out, err := exec.Command(os.Getenv("HELM_BIN"), "uninstall", "-n", release.Namespace, release.Name).Output()
			if err != nil {
				return err
			}
			fmt.Fprint(w, string(out))
		}
	}
	return nil
}
