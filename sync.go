package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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

	existingSubmodules, err := detectExistingGoSubmodules(repo, gopath, false)
	if err != nil {
		if fixErr := fixExistingSubmodules(repo); fixErr != nil {
			println("failed to fix existing submodules: " + fixErr.Error())
			os.Exit(1)
		}
		existingSubmodules, err = detectExistingGoSubmodules(repo, gopath, true)
		if err != nil {
			println("failed to detect existing submodules: " + err.Error())
			os.Exit(1)
		}
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

		gitConfig = exec.Command("git", "config", "--file", gitmodules, "submodule."+relRoot+".url", httpsOrigin(pkgRepo.Origin))
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

	if err := fixExistingSubmodules(repo); err != nil {
		println("failed to fix submodules: " + err.Error())
		os.Exit(1)
	}
}

func detectExistingGoSubmodules(repo string, gopath string, printErrors bool) ([]string, error) {
	srcPath := filepath.Join(gopath, "src")

	submoduleStatus := exec.Command("git", "submodule", "status", srcPath)
	submoduleStatus.Dir = repo

	if printErrors {
		submoduleStatus.Stderr = os.Stderr
	}

	statusOut, err := submoduleStatus.StdoutPipe()
	if err != nil {
		printErr(printErrors, "detectExistingGoSubmodules failed to get StdoutPipe: %s\n", err)
		return nil, err
	}

	lineScanner := bufio.NewScanner(statusOut)

	err = submoduleStatus.Start()
	if err != nil {
		printErr(printErrors, "detectExistingGoSubmodules failed to start git submodule status: %s\n", err)
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
		printErr(printErrors, "detectExistingGoSubmodules failed to wait for git submodule status: %s\n", err)
		return nil, err
	}

	return submodules, nil
}

func printErr(print bool, format string, err error) {
	if print {
		fmt.Printf(format, err)
	}
}

// Convert any "semi-submodules" into first class submodules.
// See http://stackoverflow.com/questions/4161022/git-how-to-track-untracked-content/4162672#4162672
func fixExistingSubmodules(repo string) error {
	submodules, err := getSubmodules(repo)
	if err != nil {
		return err
	}

	var lastErr error
	for _, submodule := range submodules {
		err = fixSubmodule(submodule)
		if err != nil {
			fmt.Printf("fixExistingSubmodules failed to fix submodule %s\n", submodule)
			lastErr = err
		}
	}

	return lastErr
}

func fixSubmodule(submodule string) error {
	fmt.Printf("\x1b[31mFixing submodule %s .", submodule)
	defer fmt.Println("\x1b[0m")
	rm := exec.Command("git", "rm", "--cached", "-f", submodule)
	rm.Stderr = os.Stderr
	err := rm.Run()
	if err != nil {
		return fmt.Errorf("fixSubmodule failed to remove submodule path %s from the index: %s", submodule, err)
	}
	fmt.Printf(".")

	url, err := submoduleUrl(submodule)
	if err != nil {
		return fmt.Errorf("fixSubmodule failed to determine URL of submodule %s: %s", submodule, err)
	}

	submoduleAdd := exec.Command("git", "submodule", "add", url, submodule)
	submoduleAdd.Stderr = os.Stderr
	err = submoduleAdd.Run()
	if err != nil {
		return fmt.Errorf("fixSubmodule failed to add submodule %s: %s", submodule, err)
	}
	fmt.Printf(".")

	fmt.Printf(". done.")
	return nil
}

func submoduleUrl(submodule string) (string, error) {
	submoduleQuery := exec.Command("git", "remote", "show", "origin")
	submoduleQuery.Dir = submodule
	submoduleQuery.Stderr = os.Stderr

	lsFileOut, err := submoduleQuery.StdoutPipe()
	if err != nil {
		fmt.Printf("submoduleUrl failed to get StdoutPipe: %s\n", err)
		return "", err
	}

	lineScanner := bufio.NewScanner(lsFileOut)

	err = submoduleQuery.Start()
	if err != nil {
		fmt.Printf("submoduleUrl failed to start git remote show origin: %s\n", err)
		return "", err
	}

	var url string
	for lineScanner.Scan() {
		segments := strings.Fields(lineScanner.Text())

		if len(segments) < 3 {
			continue
		}

		if segments[0] == "Fetch" && segments[1] == "URL:" {
			url = segments[2]
		}
	}

	if url == "" {
		return "", fmt.Errorf("submoduleUrl failed to find the URL of %s\n", submodule)
	}

	err = submoduleQuery.Wait()
	if err != nil {
		fmt.Printf("submoduleUrl failed to wait for git remote show origin: %s\n", err)
		return "", err
	}

	return url, nil
}

func getSubmodules(repo string) ([]string, error) {
	lsFiles := exec.Command("git", "ls-files", "--stage")
	lsFiles.Dir = repo
	lsFiles.Stderr = os.Stderr

	lsFileOut, err := lsFiles.StdoutPipe()
	if err != nil {
		fmt.Printf("getSubmodules failed to get StdoutPipe: %s\n", err)
		return nil, err
	}

	lineScanner := bufio.NewScanner(lsFileOut)

	err = lsFiles.Start()
	if err != nil {
		fmt.Printf("getSubmodules failed to start git ls-files --stage: %s\n", err)
		return nil, err
	}

	submodules := []string{}
	for lineScanner.Scan() {
		segments := strings.Fields(lineScanner.Text())

		if len(segments) < 4 {
			return nil, fmt.Errorf("invalid git ls-files output: %q", lineScanner.Text())
		}

		if segments[0] == "160000" {
			submodules = append(submodules, segments[3])
		}
	}

	err = lsFiles.Wait()
	if err != nil {
		fmt.Printf("getSubmodules failed to wait for git ls-files --stage: %s\n", err)
		return nil, err
	}

	return submodules, nil
}

var sshGitURIRegexp = regexp.MustCompile(`(git@github.com:|https?://github.com/)([^/]+)/(.*?)(\.git)?$`)

func httpsOrigin(uri string) string {
	return sshGitURIRegexp.ReplaceAllString(uri, "https://github.com/$2/$3")
}
