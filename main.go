package main

import (
	"flag"
	"os"

	"github.com/codegangsta/cli"
)

var gitModules = flag.String(
	"m",
	"./.gitmodules",
	"path to .gitmodules file to reconfigure",
)

var gopath = flag.String(
	"p",
	".",
	"path to $GOPATH to sync",
)

func main() {
	app := cli.NewApp()
	app.Name = "gosub"
	app.Usage = "go dependency submodule automator"
	app.Version = "0.0.1"

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "git-modules, m",
			Value: "./gitmodules",
		},
		cli.StringFlag{
			Name:  "gopath, p",
			Value: ".",
		},
		cli.StringSliceFlag{
			Name:  "app, a",
			Value: &cli.StringSlice{},
		},
		cli.StringSliceFlag{
			Name:  "test, t",
			Value: &cli.StringSlice{},
		},
	}

	app.Action = sync

	app.Run(os.Args)
}
