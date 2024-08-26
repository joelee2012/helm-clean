/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"fmt"

	"github.com/joelee2012/helm-clean/cmd"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

type BuildInfo struct {
	Version   string
	Commit    string
	BuildDate string
}

func main() {
	cmd.Execute(fmt.Sprintf("%#v", BuildInfo{Version: version, Commit: commit, BuildDate: date}))
}
