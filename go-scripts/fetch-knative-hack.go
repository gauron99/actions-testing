package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"text/template"
	"time"

	github "github.com/google/go-github/v68/github"
)

var (
	// search for these variables
	knSrvPrefix = "knative_serving_version="
	knEvtPrefix = "knative_eventing_version="
	knCtrPrefix = "contour_version="

	file     = "hack/ib.sh"
	fileJson = "hack/versions.json"
)

// all the components that are kept up to date
type Versions struct {
	Serving  string
	Eventing string
	Contour  string
}

const versionsScriptTemplate = `#!/usr/bin/env bash

# AUTOGENERATED FILE

set_versions() {
	knative_serving_version="{{.Serving}}"
	knative_eventing_version="{{.Eventing}}"
	contour_version="{{.Contour}}"
}
`

// entry function -- essentially "func main() for this file"
func main() {
	prTitle := "chore: Update components' versions to latest"
	ctx := context.Background()
	client := github.NewClient(nil).WithAuthToken(os.Getenv("GITHUB_TOKEN"))

	e, err := prExists(ctx, client, prTitle)
	if err != nil {
		os.Exit(1)
	}
	if e {
		fmt.Printf("PR already exists, nothing to do, exiting")
		os.Exit(0)
	}

	projects := []struct {
		owner, repo string
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

	// Get current versions used.
	v, err := readVersions(fileJson)
	if err != nil {
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
		updated = updateVersion(&v, p.repo, newV)
	}

	if !updated {
		// nothing to update
		fmt.Printf("all good, no newer component releases, exiting\n")
		os.Exit(0)
	}
	// overwrite the .json file with new latest versions
	writeFiles(v, file, fileJson)

	fmt.Printf("file %s updated! Creating a PR...\n", file)
	// create, PR etc etc

	branchName := "update-components" + time.Now().Format(time.DateOnly)
	err = prepareBranch(branchName)
	if err != nil {
		os.Exit(1)
	}
	err = createPR(ctx, client, prTitle, branchName)
	if err != nil {
		os.Exit(1)
	}
}

// get latest version of owner/repo via GH API
func getLatestVersion(ctx context.Context, client *github.Client, owner string, repo string) (v string, err error) {
	fmt.Printf("> get latest '%s/%s'...", owner, repo)
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
		return "", fmt.Errorf("internal error: returned latest release name is empty for '%s'", repo)
	}
	fmt.Println("done")
	return v, nil
}

// Read versions from a .json file
func readVersions(filename string) (v Versions, err error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return
	}
	err = json.Unmarshal(data, &v)
	return
}

// attempt to update 'v' Versions to new 'val' for specific 'repo'
func updateVersion(v *Versions, repo, val string) (updated bool) {
	if repo == "serving" && v.Serving != val {
		v.Serving = val
		updated = true
	} else if repo == "eventing" && v.Eventing != val {
		v.Eventing = val
		updated = true
	} else if repo == "net-contour" && v.Contour != val {
		v.Contour = val
		updated = true
	}
	return
}

// Overwrite both:
// (1) .json which holds the versions
// (2) autogenerated .sh script files
func writeFiles(v Versions, fileScript, fileJson string) error {
	// write to json file
	err := writeVersionsSource(v, fileJson)
	if err != nil {
		return fmt.Errorf("failed to write to json: %v", err)
	}
	// write to script file
	err = writeVersionsScript(v, fileScript)
	if err != nil {
		return fmt.Errorf("failed to generate script: %v", err)
	}
	return nil
}

// write .json file with updated versions
func writeVersionsSource(v Versions, file string) error {
	vB, err := json.MarshalIndent(v, "", "	")
	if err != nil {
		return fmt.Errorf("cant Marshal versions: %v", err)
	}
	f, err := os.Create(file)
	if err != nil {
		return err
	}
	_, err = f.Write(vB)
	return err
}

func writeVersionsScript(v Versions, filename string) error {
	tmpl, err := template.New("versions").Parse(versionsScriptTemplate)
	if err != nil {
		return err
	}
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	if err := tmpl.Execute(f, v); err != nil {
		return err
	}
	return nil
}

// prepare branch for PR via git commands
func prepareBranch(branchName string) error {
	fmt.Print("> prepare branch...")
	cmd := exec.Command("bash", "-c", fmt.Sprintf(`
		git config --local user.email "david.fridrich19@gmail.com" &&
		git config --local user.name "David Fridrich" &&
		git switch -c %s &&
		git add %s &&
		git commit -m "update components" &&
		git push --set-upstream origin %s
	`, branchName, file, branchName))

	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	o, err := cmd.CombinedOutput()
	fmt.Printf("> %s\n", o)
	fmt.Println("ready")
	return err
}

// create a PR for the new updates
func createPR(ctx context.Context, client *github.Client, title string, branchName string) error {
	fmt.Print("> creating PR...")
	body := fmt.Sprintf("%s\n/assign @gauron99", title)

	newPR := github.NewPullRequest{
		Title:               github.Ptr(title),
		Base:                github.Ptr("main"),
		Head:                github.Ptr(branchName),
		Body:                github.Ptr(body),
		MaintainerCanModify: github.Ptr(true),
	}
	pr, _, err := client.PullRequests.Create(ctx, "gauron99", "actions-testing", &newPR)
	if err != nil {
		fmt.Printf("PR looks like this:\n%#v\n", pr)
		fmt.Printf("err: %s\n", err)
		return err
	}
	fmt.Println("ready")
	return nil
}

// returns true when PR with given title already exists in knative/func repo
// otherwise false. Returns an error if occured, otherwise nil.
func prExists(ctx context.Context, c *github.Client, title string) (bool, error) {
	opt := &github.PullRequestListOptions{State: "open"}
	list, _, err := c.PullRequests.List(ctx, "gauron99", "actions-testing", opt)
	if err != nil {
		return false, fmt.Errorf("errror pulling PRs in knative/func: %s", err)
	}
	for _, pr := range list {
		if pr.GetTitle() == title {
			// gauron99 - currently cannot update already existing PR, shouldnt happen
			return true, nil
		}
	}
	return false, nil
}
