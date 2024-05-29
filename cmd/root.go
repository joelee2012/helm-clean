package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"time"

	str2duration "github.com/xhit/go-str2duration/v2"

	"github.com/spf13/cobra"
)

func newRootCmd() *cobra.Command {
	var clean = Clean{}
	var rootCmd = &cobra.Command{
		Use:     "clean",
		Short:   "A helm plugin to clean release by date",
		Long:    `A helm plugin to clean release by date`,
		Version: "1.0.0",
		RunE: func(cmd *cobra.Command, args []string) error {
			return clean.Run()
		},
	}
	rootCmd.Flags().StringVarP(&clean.Before, "before", "b", "30d", "The last updated time before now, eg: 3d4h")
	rootCmd.Flags().StringVarP(&clean.Filter, "filter", "f", ".*", "A regular expression, The chart of releases that match the expression will be included in the results")
	rootCmd.Flags().BoolVarP(&clean.DryRun, "dry-run", "d", true, "Dry run mode only print the release info")
	return rootCmd
}

func Execute() {
	err := newRootCmd().Execute()
	if err != nil {
		os.Exit(1)
	}
}

type Clean struct {
	Before string
	DryRun bool
	Filter string
}

type Release struct {
	Version                                           string `json:"app_version"`
	Chart, Name, Namespace, Status, Updated, Revision string
}

type ReleaseList []*Release

var timeFormat = "2006-01-02T15:04:05"

func (c *Clean) ListRelease() (ReleaseList, error) {
	duration, err := str2duration.ParseDuration(c.Before)
	if err != nil {
		return nil, err
	}
	args := []string{"list", "--no-headers", "-o", "json", "--time-format", timeFormat}
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
	for _, release := range rList {
		t, err := time.Parse(timeFormat, release.Updated)
		if err != nil {
			return nil, err
		}
		newTime := t.Add(duration)
		if now.After(newTime) && pattern.MatchString(release.Chart) {
			result = append(result, release)
		}
	}
	return result, nil
}

func (c *Clean) Run() error {
	rList, err := c.ListRelease()
	if err != nil {
		return err
	}

	for _, release := range rList {
		if c.DryRun {
			fmt.Printf("%s, %s, %s, %s, %s\n", release.Namespace, release.Name, release.Updated, release.Chart, release.Version)
		} else {
			out, err := exec.Command(os.Getenv("HELM_BIN"), "uninstall", "-n", release.Namespace, release.Name).Output()
			if err != nil {
				return err
			}
			fmt.Println(out)
		}
	}
	return nil

}
