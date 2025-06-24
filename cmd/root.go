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
	var opts = CleanOpts{}
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
			opts.Run(os.Stdout)
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			output, _ := cmd.Flags().GetString("output")

			if !slices.Contains(formats, output) {
				return fmt.Errorf("invalid format type [%s] for [-o, --output] flag", output)
			}
			return nil
		},
	}
	rootCmd.Flags().DurationVarP(&opts.Before, "before", "b", 0, "The last updated time before now, eg: 8h, (default 0) equal run 'helm list'")
	rootCmd.Flags().BoolVarP(&opts.DryRun, "dry-run", "d", true, "Dry run mode only print the release info")
	rootCmd.Flags().BoolVarP(&opts.AllNamespace, "all-namespaces", "A", false, "Check releases across all namespaces")
	rootCmd.Flags().StringVarP(&opts.Output, "output", "o", "table", "prints the output in the specified format. Allowed values: table, csv")
	rootCmd.Flags().StringSliceVarP(&opts.IncludeChart, "include-chart", "I", []string{}, `Regular expression, the chart of releases that matched the
expression will be included in the result only (can specify multiple)`)
	rootCmd.Flags().StringSliceVarP(&opts.ExcludeChart, "exclude-chart", "E", []string{}, `Regular expression, the chart of releases that matched the
expression will be excluded from the result (can specify multiple)`)
	rootCmd.Flags().StringSliceVarP(&opts.Include, "include", "i", []string{}, `Regular expression '<namespace>:<release>', the matched
release and namespace will be included in result only (can specify multiple)`)
	rootCmd.Flags().StringSliceVarP(&opts.Exclude, "exclude", "e", []string{}, `Regular expression '<namespace>:<release>', the matched 
release and namespace will be excluded from the result (can specify multiple)`)
	rootCmd.Flags().IntVarP(&opts.Max, "max", "m", 256, "maximum number of releases to fetch (default 256)")
	return rootCmd
}

func Execute(version string) {
	err := newRootCmd(version).Execute()
	if err != nil {
		os.Exit(1)
	}
}

type CleanOpts struct {
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

var timeFormat = "2006-01-02 15:04:05.999999999 -0700 MST"

func RunHelmCmd(args ...string) (*bytes.Buffer, error) {
	helm := os.Getenv("HELM_BIN")
	if helm == "" {
		return nil, fmt.Errorf("require environment variable HELM_BIN, but not found")
	}
	cmd := exec.Command(helm, args...)
	var stderr, stdout bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("%v: %s", err, stderr.String())
	}
	return &stdout, nil
}
func (c *CleanOpts) ListRelease() (RList, error) {
	args := []string{"list", "--no-headers", "-o", "json"}
	if c.AllNamespace {
		args = append(args, "-A")
	}
	if c.Max != 256 {
		args = append(args, "-m", strconv.Itoa(c.Max))
	}

	stdout, err := RunHelmCmd(args...)
	if err != nil {
		return nil, err
	}
	var rList RList
	cobra.CheckErr(json.NewDecoder(stdout).Decode(&rList))
	now := time.Now()
	includeChart := regexp.MustCompile(strings.Join(c.IncludeChart, "|"))
	excludeChart := regexp.MustCompile(strings.Join(c.ExcludeChart, "|"))
	checkExcludeChart := len(c.ExcludeChart) > 0
	include := regexp.MustCompile(strings.Join(c.Include, "|"))
	exclude := regexp.MustCompile(strings.Join(c.Exclude, "|"))
	checkExclude := len(c.Exclude) > 0

	var result RList
	for _, r := range rList {
		updateTime, err := time.Parse(timeFormat, r.Updated)
		if err != nil {
			return nil, err
		}
		rn := fmt.Sprintf("%s:%s", r.Namespace, r.Name)
		if now.After(updateTime.Add(c.Before)) && includeChart.MatchString(r.Chart) && include.MatchString(rn) {
			if (checkExclude && exclude.MatchString(rn)) || (checkExcludeChart && excludeChart.MatchString(r.Chart)) {
				continue
			}
			result = append(result, r)
		}
	}
	return result, nil
}

func (c *CleanOpts) Run(w io.Writer) {
	rList, err := c.ListRelease()
	cobra.CheckErr(err)
	if c.DryRun {
		t := table.NewWriter()
		t.SetOutputMirror(w)
		t.AppendHeader(table.Row{"NAME", "NAMESPACE", "REVISION", "UPDATED", "STATUS", "CHART", "APP VERSION"})
		for _, r := range rList {
			t.AppendRow(table.Row{r.Name, r.Namespace, r.Revision, r.Updated, r.Status, r.Chart, r.AppVersion})
		}
		t.SortBy([]table.SortBy{{Name: "NAME", Mode: table.Asc}, {Name: "NAMESPACE", Mode: table.Asc}})
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
			out, err := RunHelmCmd("uninstall", "-n", release.Namespace, release.Name)
			cobra.CheckErr(err)
			fmt.Fprint(w, out.String())
		}
	}
}
