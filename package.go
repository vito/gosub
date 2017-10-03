package main

import (
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"sort"
)

const packageBatchSize = 100

// via go list -json
type Package struct {
	Standard   bool
	ImportPath string

	Deps []string

	TestImports  []string
	XTestImports []string
}

func listPackages(pkgs map[string]Package, goos string, ps ...string) error {
	if len(ps) == 0 {
		return nil
	}

	packages := []string{}
	remainingPackages := ps
	for {
		if len(remainingPackages) == 0 {
			break
		}

		if len(remainingPackages) < packageBatchSize {
			packages = remainingPackages
			remainingPackages = nil
		} else {
			packages = remainingPackages[:packageBatchSize]
			remainingPackages = remainingPackages[packageBatchSize:]
		}

		listPackages := exec.Command(
			"go",
			append([]string{"list", "-e", "-json"}, packages...)...,
		)
		listPackages.Env = []string{"GOOS=" + goos, "GOPATH=" + os.Getenv("GOPATH")}

		listPackages.Stderr = os.Stderr

		packageStream, err := listPackages.StdoutPipe()
		if err != nil {
			return err
		}

		err = listPackages.Start()
		if err != nil {
			return err
		}

		decoder := json.NewDecoder(packageStream)

		for {
			var pkg Package
			err := decoder.Decode(&pkg)
			if err != nil {
				if err == io.EOF {
					break
				}

				return err
			}

			if existing, exists := pkgs[pkg.ImportPath]; exists {
				for _, dep := range pkg.Deps {
					hasDep := false
					for _, existingDep := range existing.Deps {
						if dep == existingDep {
							hasDep = true
							break
						}
					}

					if !hasDep {
						existing.Deps = append(existing.Deps, dep)
						pkgs[pkg.ImportPath] = existing
					}
				}
			} else {
				pkgs[pkg.ImportPath] = pkg
			}
		}

		err = listPackages.Wait()
		if err != nil {
			return err
		}
	}

	return nil
}

type byImportPath []Package

func (ps byImportPath) Len() int               { return len(ps) }
func (ps byImportPath) Less(i int, j int) bool { return ps[i].ImportPath < ps[j].ImportPath }
func (ps byImportPath) Swap(i int, j int)      { ps[i], ps[j] = ps[j], ps[i] }

func listAllPlatformPackages(packages ...string) ([]Package, error) {
	allPackages := map[string]Package{}

	for _, goos := range []string{"linux", "darwin", "windows"} {
		err := listPackages(allPackages, goos, packages...)
		if err != nil {
			return nil, err
		}
	}

	pkgList := []Package{}
	for _, pkg := range allPackages {
		pkgList = append(pkgList, pkg)
	}

	sort.Sort(byImportPath(pkgList))

	return pkgList, nil
}

func getAppImports(packages ...string) ([]string, error) {
	appPackages, err := listAllPlatformPackages(packages...)
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
	testPackages, err := listAllPlatformPackages(packages...)
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
	pkgs, err := listAllPlatformPackages(packages...)
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
	pkgs, err := listAllPlatformPackages(packages...)
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
