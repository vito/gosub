package main

import (
	"fmt"

	"github.com/codegangsta/cli"
)

func list(c *cli.Context) error {
	appPackages := c.StringSlice("app")
	testPackages := c.StringSlice("test")

	appImports, err := getAppImports(appPackages...)
	if err != nil {
		return fmt.Errorf("failed to detect app imports: " + err.Error())
	}

	testImports, err := getTestImports(testPackages...)
	if err != nil {
		return fmt.Errorf("failed to detect test imports: " + err.Error())
	}

	allImports := append(appImports, testImports...)

	allDeps, err := getAllDeps(allImports...)
	if err != nil {
		return fmt.Errorf("failed to get deps: " + err.Error())
	}

	deps, err := filterNonStandard(allDeps...)
	if err != nil {
		return fmt.Errorf("failed to filter deps: " + err.Error())
	}

	for _, dep := range deps {
		fmt.Println(dep)
	}

	return nil
}
