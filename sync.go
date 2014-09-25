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

	gitmodules := filepath.Join(repo, ".gitmodules")

	submodulesToRemove := map[string]bool{}
	for _, submodule := range existingSubmodules {
		submodulesToRemove[submodule] = true
	}

	for pkgRoot, pkgRepo := range pkgRoots {
		relRoot, err := filepath.Rel(absRepo, pkgRoot)
		if err != nil {
			println("could not resolve submodule: " + err.Error())
			os.Exit(1)
		}

		fmt.Println("\x1b[32msyncing " + relRoot + "\x1b[0m")

		// keep this submodule
		delete(submodulesToRemove, relRoot)

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

		status := exec.Command("git", "status", "--porcelain")
		status.Dir = filepath.Join(absRepo, relRoot)

		statusOutput, err := status.Output()
		if err != nil {
			println("error fetching submodule status: " + err.Error())
			os.Exit(1)
		}

		if len(statusOutput) != 0 {
			println("\x1b[31msubmodule is dirty: " + pkgRoot + "\x1b[0m")
			os.Exit(1)
		}

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

		gitAdd := exec.Command("git", "add", gitmodules)
		gitAdd.Dir = repo
		gitAdd.Stderr = os.Stderr

		err = gitAdd.Run()
		if err != nil {
			println("error staging submodule config: " + err.Error())
			os.Exit(1)
		}
	}

	for submodule, _ := range submodulesToRemove {
		fmt.Println("\x1b[31mremoving " + submodule + "\x1b[0m")

		rm := exec.Command("git", "rm", "--cached", "-f", submodule)
		rm.Dir = repo
		rm.Stderr = os.Stderr

		err := rm.Run()
		if err != nil {
			println("error clearing submodule: " + err.Error())
			os.Exit(1)
		}

		gitConfig := exec.Command("git", "config", "--file", gitmodules, "--remove-section", "submodule."+submodule)
		gitConfig.Dir = repo
		gitConfig.Stderr = os.Stderr

		err = gitConfig.Run()
		if err != nil {
			println("error removing submodule config: " + err.Error())
			os.Exit(1)
		}

		gitAdd := exec.Command("git", "add", gitmodules)
		gitAdd.Dir = repo
		gitAdd.Stderr = os.Stderr

		err = gitAdd.Run()
		if err != nil {
			println("error staging submodule config: " + err.Error())
			os.Exit(1)
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
		segments := strings.Split(lineScanner.Text()[1:], " ")

		if len(segments) < 2 {
			return nil, fmt.Errorf("invalid git status output: %q", lineScanner.Text())
		}

		submodules = append(submodules, segments[1])
	}

	err = submoduleStatus.Wait()
	if err != nil {
		return nil, err
	}

	return submodules, nil
}
