package main

import (
	"encoding/json"
	"io"
	"os"
	"os/exec"
)

// via go list -json
type Package struct {
	Standard   bool
	ImportPath string

	Deps []string

	TestImports  []string
	XTestImports []string
}

func listPackages(packages ...string) ([]Package, error) {
	if len(packages) == 0 {
		return []Package{}, nil
	}

	listPackages := exec.Command(
		"go",
		append([]string{"list", "-e", "-json"}, packages...)...,
	)

	listPackages.Stderr = os.Stderr

	packageStream, err := listPackages.StdoutPipe()
	if err != nil {
		return nil, err
	}

	err = listPackages.Start()
	if err != nil {
		return nil, err
	}

	decoder := json.NewDecoder(packageStream)

	pkgs := []Package{}
	for {
		var pkg Package
		err := decoder.Decode(&pkg)
		if err != nil {
			if err == io.EOF {
				break
			}

			return nil, err
		}

		pkgs = append(pkgs, pkg)
	}

	err = listPackages.Wait()
	if err != nil {
		return nil, err
	}

	return pkgs, nil
}

func getAppImports(packages ...string) ([]string, error) {
	appPackages, err := listPackages(packages...)
	if err != nil {
		return nil, err
	}

	imports := []string{}
	for _, pkg := range appPackages {
		imports = append(imports, pkg.ImportPath)
	}

	return imports, nil
}

func getTestImports(packages ...string) ([]string, error) {
	testPackages, err := listPackages(packages...)
	if err != nil {
		return nil, err
	}

	imports := []string{}
	imports = append(imports, packages...)

	for _, pkg := range testPackages {
		imports = append(imports, pkg.TestImports...)
		imports = append(imports, pkg.XTestImports...)
	}

	return filterNonStandard(imports...)
}

func getAllDeps(packages ...string) ([]string, error) {
	pkgs, err := listPackages(packages...)
	if err != nil {
		return nil, err
	}

	allDeps := []string{}
	allDeps = append(allDeps, packages...)

	for _, pkg := range pkgs {
		if pkg.Standard {
			continue
		}

		allDeps = append(allDeps, pkg.Deps...)
	}

	return allDeps, nil
}

func filterNonStandard(packages ...string) ([]string, error) {
	pkgs, err := listPackages(packages...)
	if err != nil {
		return nil, err
	}

	nonStandard := []string{}
	for _, pkg := range pkgs {
		if pkg.Standard {
			continue
		}

		nonStandard = append(nonStandard, pkg.ImportPath)
	}

	return nonStandard, nil
}
