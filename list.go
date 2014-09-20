package main

import (
	"fmt"
	"os"

	"github.com/codegangsta/cli"
)

func list(c *cli.Context) {
	appPackages := c.StringSlice("app")
	testPackages := c.StringSlice("test")

	appImports, err := getAppImports(appPackages...)
	if err != nil {
		println("failed to detect app imports: " + err.Error())
		os.Exit(1)
	}

	testImports, err := getTestImports(testPackages...)
	if err != nil {
		println("failed to detect test imports: " + err.Error())
		os.Exit(1)
	}

	allImports := append(appImports, testImports...)

	allDeps, err := getAllDeps(allImports...)
	if err != nil {
		println("failed to get deps: " + err.Error())
		os.Exit(1)
	}

	deps, err := filterNonStandard(allDeps...)
	if err != nil {
		println("failed to filter deps: " + err.Error())
		os.Exit(1)
	}

	for _, dep := range deps {
		fmt.Println(dep)
	}
}
