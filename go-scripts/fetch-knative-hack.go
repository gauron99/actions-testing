package main

import (
	"context"
	"fmt"
	"os"

	github "github.com/google/go-github/v68/github"
)

func main() {
	ctx := context.TODO()
	client := github.NewClient(nil).WithAuthToken(os.Getenv("GITHUB_TOKEN"))
	c, r, err := client.Repositories.GetLatestRelease(ctx, "knative", "serving")
	if err != nil {
		fmt.Printf("chybka: %v\n", err)
		os.Exit(1)
	}
	if r.StatusCode < 200 && r.StatusCode > 299 {
		fmt.Printf("status code is %d\n", r.StatusCode)
		os.Exit(1)
	}
	fmt.Println("good")

	fmt.Printf("content: %#v\n", c)

}
