package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/codegangsta/cli"
)

func sync(c *cli.Context) {
	repo := c.String("repo")
	gopath := c.String("gopath")

	absRepo, err := filepath.Abs(repo)
	if err != nil {
		println("could not resolve repo: " + err.Error())
		os.Exit(1)
	}

	absGopath, err := filepath.Abs(gopath)
	if err != nil {
		println("could not resolve gopath: " + err.Error())
		os.Exit(1)
	}

	pkgRoots := map[string]*Repo{}

	for _, dep := range c.Args() {
		root, repo, err := getDepRoot(absRepo, absGopath, dep)
		if err != nil {
			println("failed to get dependency repo: " + err.Error())
			os.Exit(1)
		}

		pkgRoots[root] = repo
	}

	existingSubmodules, err := detectExistingGoSubmodules(repo, gopath)
	if err != nil {
		println("failed to detect existing submodules: " + err.Error())
		os.Exit(1)
	}

	for _, submodule := range existingSubmodules {
		rm := exec.Command("git", "rm", "--cached", submodule)
		rm.Dir = repo
		rm.Stderr = os.Stderr

		err := rm.Run()
		if err != nil {
			println("error clearing submodule: " + err.Error())
			os.Exit(1)
		}
	}

	for pkgRoot, pkgRepo := range pkgRoots {
		relRoot, err := filepath.Rel(absRepo, pkgRoot)
		if err != nil {
			println("could not resolve submodule: " + err.Error())
			os.Exit(1)
		}

		fmt.Println(relRoot)

		add := exec.Command("git", "add", pkgRoot)
		add.Dir = repo
		add.Stderr = os.Stderr

		err = add.Run()
		if err != nil {
			println("error clearing submodule: " + err.Error())
			os.Exit(1)
		}

		if pkgRepo == nil {
			// non-git dependency; vendored
			continue
		}

		gitmodules := filepath.Join(repo, ".gitmodules")

		gitConfig := exec.Command("git", "config", "--file", gitmodules, "submodule."+relRoot+".path", relRoot)
		gitConfig.Stderr = os.Stderr

		err = gitConfig.Run()
		if err != nil {
			println("error configuring submodule: " + err.Error())
			os.Exit(1)
		}

		gitConfig = exec.Command("git", "config", "--file", gitmodules, "submodule."+relRoot+".url", pkgRepo.Origin)
		gitConfig.Stderr = os.Stderr

		err = gitConfig.Run()
		if err != nil {
			println("error configuring submodule: " + err.Error())
			os.Exit(1)
		}

		if pkgRepo.Branch != "" {
			gitConfig = exec.Command("git", "config", "--file", gitmodules, "submodule."+relRoot+".branch", pkgRepo.Branch)
			gitConfig.Stderr = os.Stderr

			err = gitConfig.Run()
			if err != nil {
				println("error configuring submodule: " + err.Error())
				os.Exit(1)
			}
		}
	}
}

func detectExistingGoSubmodules(repo string, gopath string) ([]string, error) {
	srcPath := filepath.Join(gopath, "src")

	submoduleStatus := exec.Command("git", "submodule", "status", srcPath)
	submoduleStatus.Dir = repo

	submoduleStatus.Stderr = os.Stderr

	statusOut, err := submoduleStatus.StdoutPipe()
	if err != nil {
		return nil, err
	}

	lineScanner := bufio.NewScanner(statusOut)

	err = submoduleStatus.Start()
	if err != nil {
		return nil, err
	}

	submodules := []string{}
	for lineScanner.Scan() {
		segments := strings.Split(lineScanner.Text(), " ")

		if len(segments) < 3 {
			return nil, fmt.Errorf("invalid git status output: %q", lineScanner.Text())
		}

		submodules = append(submodules, segments[2])
	}

	err = submoduleStatus.Wait()
	if err != nil {
		return nil, err
	}

	return submodules, nil
}
