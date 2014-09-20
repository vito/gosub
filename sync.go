package main

import (
	"fmt"
	"os"

	"github.com/codegangsta/cli"
)

// via go list -json
type Package struct {
	Standard   bool
	ImportPath string

	Deps []string

	TestImports  []string
	XTestImports []string
}

func sync(c *cli.Context) {
	appPackages := c.StringSlice("app")
	testPackages := c.StringSlice("test")

	appImports, err := getAppImports(appPackages...)
	if err != nil {
		println("failed to detect app imports: " + err.Error())
		os.Exit(1)
	}

	testImports, err := getTestImports(testPackages...)
	if err != nil {
		println("failed to detect test imports for: " + err.Error())
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
	// add each app package to a list of dependencies,
	// for each test package, add its TestImports and XTestImports to the list,
	// get the dependencies for each package on the list,
	// remove all gopath entries from .gitmodules,
	// and add the new set of gopath dependencies
}
