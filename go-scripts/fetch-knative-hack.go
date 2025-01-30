package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	github "github.com/google/go-github/v68/github"
)

func getVersionFromFile() (srv string, evt string, err error) {
	f := "go-scripts/ib.sh"
	readFile, err := os.Open(f)
	if err != nil {
		err = fmt.Errorf("cant open file '%s': %v", f, err)
	}
	defer readFile.Close()

	fs := bufio.NewScanner(readFile)
	fs.Split(bufio.ScanLines)
	for fs.Scan() {
		line := fs.Text()
		if strings.HasPrefix(line, "local kn_serving") {
			srv = "v" + strings.Split(line, "=")[1]
		} else if strings.HasPrefix(line, "local kn_eventing") {
			evt = "v" + strings.Split(line, "=")[1]
		}
		if srv != "" && evt != "" {
			break
		}
	}
	return
}

func main() {
	ctx := context.TODO()
	client := github.NewClient(nil).WithAuthToken(os.Getenv("GITHUB_TOKEN"))
	srvCl, r, err := client.Repositories.GetLatestRelease(ctx, "knative", "serving")
	if err != nil {
		fmt.Printf("chybka: %v\n", err)
		os.Exit(1)
	}
	if r.StatusCode < 200 && r.StatusCode > 299 {
		fmt.Printf("status code is %d\n", r.StatusCode)
		os.Exit(1)
	}
	srv, evt, err := getVersionFromFile()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("current version in file: srv=%s, evt=%s\n", srv, evt)

	if srv == *srvCl.Name {
		fmt.Println("Serving matches")
	}
	// if evt == *evtCl.Name {
	// 	fmt.Println("eventing matches")
	// }
}
