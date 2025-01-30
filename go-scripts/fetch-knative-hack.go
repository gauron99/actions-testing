package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	github "github.com/google/go-github/v68/github"
)

var (
	// search for these variables
	knSrvPrefix = "knative_serving_version="
	knEvtPrefix = "knative_eventing_version="
	knCnrPrefix = "contour_version="
)

// get latest version of owner/repo via GH API
func getLatestVersion(ctx context.Context, client *github.Client, owner string, repo string) (v string, err error) {
	rr, res, err := client.Repositories.GetLatestRelease(ctx, owner, repo)
	if err != nil {
		err = fmt.Errorf("error: Return status code of request for latest serving release is %d", res.StatusCode)
		return
	}
	if res.StatusCode < 200 && res.StatusCode > 299 {
		err = fmt.Errorf("error: Return status code of request for latest eventing release is %d", res.StatusCode)
		return
	}
	v = *rr.Name
	return v, nil
}

// read the allocate.sh file where serving and eventing versions are
// located. Read that file to find them via prefix above. Fetch their version
// and return them in 'v1.23.0' format. (To be compared with the current latest)
func getVersionsFromFile() (srv string, evt string, cnr string, err error) {
	srv = "" //serving
	evt = "" //eventing
	cnr = "" //net-concour (knative-extensions)

	var f = "allocate.sh"

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

// Update version in file if new releases of eventing/serving/net-concour exist
// if applicable.
func tryUpdateFile(upstreams []struct{ owner, repo, version string }) (updated bool, err error) {
	file := "hack/allocate.sh"
	updated = false

	// get current versions used. Get all together to limit opening/closing
	// the file
	oldSrv, oldEvt, oldCnr, err := getVersionsFromFile()
	if err != nil {
		return false, err
	}

	// update files to latest release where applicable
	for _, upstream := range upstreams {
		var cmd *exec.Cmd
		switch upstream.repo {
		case "serving":
			if upstream.version != oldSrv {
				fmt.Printf("update serving from '%s' to '%s'\n", oldSrv, upstream.version)
				cmd = exec.Command("sed", "-i", "-e", "s/"+knSrvPrefix+oldSrv+"/"+knSrvPrefix+upstream.version+"/g", file)
			}
		case "eventing":
			if upstream.version != oldEvt {
				fmt.Printf("update eventing from '%s' to '%s'\n", oldEvt, upstream.version)
				cmd = exec.Command("sed", "-i", "-e", "s/"+knEvtPrefix+oldEvt+"/"+knEvtPrefix+upstream.version+"/g", file)
			}
		case "concour":
			if upstream.version != oldCnr {
				fmt.Printf("update concour from '%s' to '%s'\n", oldCnr, upstream.version)
				cmd = exec.Command("sed", "-i", "-e", "s/"+knCnrPrefix+oldCnr+"/"+knCnrPrefix+upstream.version+"/g", file)
			}
		default:
			err = fmt.Errorf("unkown upstream.repo in for loop, exiting")
			break
		}
		err = cmd.Run()
		if err != nil {
			return false, fmt.Errorf("failed to sed %s: %v", err, upstream.repo)
		}
		updated = true
	}

	return updated, nil
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
			repo:  "net-concour",
		},
	}
	var err error
	for i, p := range projects {
		projects[i].version, err = getLatestVersion(ctx, client, p.owner, p.repo)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error while getting latest v of %s/%s: %v\n", p.owner, p.repo, err)
			os.Exit(1)
		}
	}

	updated, err := tryUpdateFile(projects)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if !updated {
		// nothing was updated, nothing to do
		fmt.Print("nothing to update")
		os.Exit(0)
	}
	// create, PR etc etc
}
