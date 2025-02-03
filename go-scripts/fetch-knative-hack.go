package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	github "github.com/google/go-github/v68/github"
)

var (
	// search for these variables
	knSrvPrefix = "knative_serving_version="
	knEvtPrefix = "knative_eventing_version="
	knCnrPrefix = "contour_version="
)

// get current branch this is running o
// get latest version of owner/repo via GH API
func getLatestVersion(ctx context.Context, client *github.Client, owner string, repo string) (v string, err error) {
	fmt.Printf("get latest repo %s/%s\n", owner, repo)
	rr, res, err := client.Repositories.GetLatestRelease(ctx, owner, repo)
	if err != nil {
		err = fmt.Errorf("error: request for latest %s release: %v", owner+"/"+repo, err)
		return
	}
	if res.StatusCode < 200 && res.StatusCode > 299 {
		err = fmt.Errorf("error: Return status code of request for latest %s release is %d", owner+"/"+repo, res.StatusCode)
		return
	}
	v = *rr.Name
	if v == "" {
		return "", fmt.Errorf("error: returned latest release name is empty for '%s'", repo)
	}
	return v, nil
}

// read the ib.sh file where serving and eventing versions are
// located. Read that file to find them via prefix above. Fetch their version
// and return them in 'v1.23.0' format. (To be compared with the current latest)
func getVersionsFromFile() (srv string, evt string, cnr string, err error) {
	srv = "" //serving
	evt = "" //eventing
	cnr = "" //net-concour (knative-extensions)

	var f = "hack/ib.sh"

	file, err := os.OpenFile(f, os.O_RDWR, 0600)
	if err != nil {
		err = fmt.Errorf("cant open file '%s': %v", f, err)
	}
	defer file.Close()
	// read file line by line
	fs := bufio.NewScanner(file)
	fs.Split(bufio.ScanLines)
	for fs.Scan() {
		// Look for a prefix in a trimmed line.
		line := strings.TrimSpace(fs.Text())
		// Fetch only the version number (after '=' without spaces because bash)
		if strings.HasPrefix(line, knSrvPrefix) {
			srv = strings.Split(line, "=")[1]
			if !strings.HasPrefix(srv, "v") {
				srv = "v" + srv
			}
		} else if strings.HasPrefix(line, knEvtPrefix) {
			evt = strings.Split(line, "=")[1]
			if !strings.HasPrefix(evt, "v") {
				evt = "v" + evt
			}
		} else if strings.HasPrefix(line, knCnrPrefix) {
			cnr = strings.Split(line, "=")[1]
			if !strings.HasPrefix(cnr, "v") {
				cnr = "v" + cnr
			}
		}
		// if all values are acquired, no need to continue
		if srv != "" && evt != "" && cnr != "" {
			break
		}
	}
	return
}

// try updating the version of component named by "repo" via 'sed'
func tryUpdateFile(repo, newV, oldV string) (bool, error) {
	quoteWrap := func(s string) string { return "\"" + s + "\"" }
	if newV != oldV {
		fmt.Printf("Updating %s from '%s' to '%s'\n", repo, oldV, newV)
		cmd := exec.Command("sed", "-i", "-e", "s/"+knSrvPrefix+quoteWrap(oldV)+"/"+knSrvPrefix+quoteWrap(newV)+"/g", file)
		err := cmd.Run()
		if err != nil {
			return false, fmt.Errorf("error while updating '%s' version: %s", repo, err)
		}
		return true, nil
	}
	return false, nil
}

func prepareBranch() error {
	branchName := "update-components" + time.Now().Format(time.DateOnly)
	cmd := exec.Command(
		"git", "config", "user.email", "fridrich.david19@gmail.com", "&&",
		"git", "config", "user.name", "David Fridrich(bot)", "&&",
		"git", "checkout", "-b", branchName, "&&",
		"git", "add", "hack/ib.sh",
		"git", "commit", "-m", "update components", "&&",
		"git", "push", "--set-upstream", "origin", branchName)
	out, err := cmd.Output()
	if err != nil {
		return err
	}
	fmt.Printf("out: %s\n", out)
	return nil
}

// create a PR for the new updates
func createPR(ctx context.Context, client *github.Client, title string) error {
	newPR := github.NewPullRequest{Title: github.Ptr(title), MaintainerCanModify: github.Ptr(true)}
	client.PullRequests.Create(ctx, "gauron99", "actions-testing", &newPR)
	return nil
}

// MAIN
func main() {
	ctx := context.Background()
	client := github.NewClient(nil).WithAuthToken(os.Getenv("GITHUB_TOKEN"))

	// PR already exists?
	// TODO

	projects := []struct {
		owner, repo, version string
	}{
		{
			owner: "knative",
			repo:  "serving",
		},
		{
			owner: "knative",
			repo:  "eventing",
		},
		{
			owner: "knative-extensions",
			repo:  "net-contour",
		},
	}
	// get current versions used. Get all together to limit opening/closing
	// the file
	oldSrv, oldEvt, oldCntr, err := getVersionsFromFile()
	if err != nil {
		fmt.Printf("err: %w\n", err)
		os.Exit(1)
	}

	updated := false
	// cycle through all versions of components listed above, fetch their
	// latest from github releases - cmp them - create PR for update if necessary
	for _, p := range projects {
		newV, err := getLatestVersion(ctx, client, p.owner, p.repo)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error while getting latest v of %s/%s: %v\n", p.owner, p.repo, err)
			os.Exit(1)
		}

		// sync the old repo & version
		oldV := ""
		switch p.repo {
		case "serving":
			oldV = oldSrv
		case "eventing":
			oldV = oldEvt
		case "net-contour":
			oldV = oldCntr
		}
		// check if component is eligible for update & update if possible
		isNew, err := tryUpdateFile(p.repo, newV, oldV)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		// if any of the files are updated, set this so we create a PR later
		if isNew {
			updated = true
		}
	}

	if !updated {
		// nothing was updated, nothing to do
		fmt.Printf("all good, no newer component releases, exiting\n")
		os.Exit(1)
	}
	fmt.Printf("file %s updated! Creating a PR...\n", "hack/ib.sh")
	// create, PR etc etc
	err = prepareBranch()

	file := "hack/ib.sh"
	prTitle := fmt.Sprintf("chore: testing PR, trying to update a %s file", file)
	err = createPR(ctx, client, prTitle)
}
