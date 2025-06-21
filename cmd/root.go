package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"

	"github.com/spf13/cobra"
)

var formats = []string{"csv", "table"}

func newRootCmd(version string) *cobra.Command {
	var clean = Clean{}
	var rootCmd = &cobra.Command{
		Use:   "clean",
		Short: "A helm plugin to list/clean out of date releases",
		Long: `A helm plugin to list/clean out of date releases
		
List/Clean the release which was not updated in duration

Examples:
	# List all release which was not updated in 240h
	helm clean -A -b 240h

	# List release which was created by chart that matched chart-1
	helm clean -A -b 240h -I chart-1

	# List release was not created by chart that matched chart-1
	helm clean -A -b 240h -E chart-1

	# Exclude namespace match pattern
	helm clean -A -b 240h -e kube-system

	# Exclude release match pattern
	helm clean -A -b 240h -e ':release-1'

	# Exclude release and namespace match pattern
	helm clean -A -b 240h -e '.*-namespace:.*-release'
`,
		Version: version,
		Run: func(cmd *cobra.Command, args []string) {
			clean.Run(os.Stdout)
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			output, _ := cmd.Flags().GetString("output")

			if !slices.Contains(formats, output) {
				return fmt.Errorf("invalid format type [%s] for [-o, --output] flag", output)
			}
			return nil
		},
	}
	rootCmd.Flags().DurationVarP(&clean.Before, "before", "b", 0, "The last updated time before now, eg: 8h, (default 0) equal run 'helm list'")
	rootCmd.Flags().BoolVarP(&clean.DryRun, "dry-run", "d", true, "Dry run mode only print the release info")
	rootCmd.Flags().BoolVarP(&clean.AllNamespace, "all-namespaces", "A", false, "Check releases across all namespaces")
	rootCmd.Flags().StringVarP(&clean.Output, "output", "o", "table", "prints the output in the specified format. Allowed values: table, csv")
	rootCmd.Flags().StringSliceVarP(&clean.IncludeChart, "include-chart", "I", []string{}, `Regular expression, the chart of releases that matched the
expression will be included in the result only (can specify multiple)`)
	rootCmd.Flags().StringSliceVarP(&clean.ExcludeChart, "exclude-chart", "E", []string{}, `Regular expression, the chart of releases that matched the
expression will be excluded from the result (can specify multiple)`)
	rootCmd.Flags().StringSliceVarP(&clean.Include, "include", "i", []string{}, `Regular expression '<namespace>:<release>', the matched
release and namespace will be included in result only (can specify multiple)`)
	rootCmd.Flags().StringSliceVarP(&clean.Exclude, "exclude", "e", []string{}, `Regular expression '<namespace>:<release>', the matched 
release and namespace will be excluded from the result (can specify multiple)`)
	rootCmd.Flags().IntVarP(&clean.Max, "max", "m", 256, "maximum number of releases to fetch (default 256)")
	return rootCmd
}

func Execute(version string) {
	err := newRootCmd(version).Execute()
	if err != nil {
		os.Exit(1)
	}
}

type Clean struct {
	Before       time.Duration
	DryRun       bool
	AllNamespace bool
	ExcludeChart []string
	IncludeChart []string
	Exclude      []string
	Include      []string
	Output       string
	Max          int
}

type Release struct {
	AppVersion string `json:"app_version"`
	Chart      string
	Name       string
	Namespace  string
	Status     string
	Updated    string
	Revision   string
}

type RList []*Release

var timeFormat = "2006-01-02 15:04:05 UTC"

func RunCmd(name string, args ...string) (RList, error) {
	cmd := exec.Command(name, args...)
	var stderr, stdout bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("command %s, %s", err, stderr.String())
	}
	var rList RList
	cobra.CheckErr(json.NewDecoder(&stdout).Decode(&rList))
	return rList, nil
}
func (c *Clean) ListRelease() (RList, error) {
	args := []string{"list", "--no-headers", "-o", "json", "--time-format", timeFormat}
	if c.AllNamespace {
		args = append(args, "-A")
	}
	if c.Max != 256 {
		args = append(args, "-m", strconv.Itoa(c.Max))
	}
	helm := os.Getenv("HELM_BIN")
	if helm == "" {
		return nil, fmt.Errorf("require environment variable HELM_BIN, but not found")
	}
	rList, err := RunCmd(os.Getenv("HELM_BIN"), args...)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	loc, err := time.LoadLocation("Local")
	if err != nil {
		return nil, err
	}

	includeChart := regexp.MustCompile(strings.Join(c.IncludeChart, "|"))
	excludeChart := regexp.MustCompile(strings.Join(c.ExcludeChart, "|"))
	checkExcludeChart := len(c.ExcludeChart) > 0
	include := regexp.MustCompile(strings.Join(c.Include, "|"))
	exclude := regexp.MustCompile(strings.Join(c.Exclude, "|"))
	checkExclude := len(c.Exclude) > 0

	var result RList
	for _, release := range rList {
		t, err := time.ParseInLocation(timeFormat, release.Updated, loc)
		if err != nil {
			return nil, err
		}
		rn := fmt.Sprintf("%s:%s", release.Namespace, release.Name)
		if now.After(t.Add(c.Before)) && includeChart.MatchString(release.Chart) && include.MatchString(rn) {
			if (checkExclude && exclude.MatchString(rn)) || (checkExcludeChart && excludeChart.MatchString(release.Chart)) {
				continue
			}
			result = append(result, release)
		}
	}
	return result, nil
}

func (c *Clean) Run(w io.Writer) {
	rList, err := c.ListRelease()
	cobra.CheckErr(err)
	if c.DryRun {
		t := table.NewWriter()
		t.SetOutputMirror(w)
		t.AppendHeader(table.Row{"NAMESPACE", "NAME", "UPDATED", "CHART", "APP VERSION"})
		for _, release := range rList {
			t.AppendRow(table.Row{release.Namespace, release.Name, release.Updated, release.Chart, release.AppVersion})
		}
		t.SortBy([]table.SortBy{{Name: "NAMESPACE", Mode: table.Asc}, {Name: "NAME", Mode: table.Asc}})
		switch c.Output {
		case "table":
			s := table.StyleLight
			s.Options = table.OptionsNoBordersAndSeparators
			t.SetStyle(s)
			t.Render()
		case "csv":
			t.RenderCSV()
		}

	} else {
		for _, release := range rList {
			out, err := exec.Command(os.Getenv("HELM_BIN"), "uninstall", "-n", release.Namespace, release.Name).CombinedOutput()
			cobra.CheckErr(err)
			fmt.Fprint(w, string(out))
		}
	}
}
