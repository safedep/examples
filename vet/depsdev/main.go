package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	v3 "deps.dev/api/v3"
	drygrpc "github.com/safedep/dry/adapters/grpc"
	"google.golang.org/grpc"
)

func main() {
	conn, err := drygrpc.GrpcClient("deps.dev",
		"api.deps.dev", "443", "", http.Header{}, []grpc.DialOption{})
	if err != nil {
		log.Fatal(err)
	}

	client := v3.NewInsightsClient(conn)
	res, err := client.GetProject(context.Background(),
		&v3.GetProjectRequest{
			ProjectKey: &v3.ProjectKey{
				Id: "github.com/psf/requests",
			},
		})

	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Project Description: %s\n", res.GetDescription())
	fmt.Printf("Project Home Page: %s\n", res.GetHomepage())
	fmt.Printf("Project License: %s\n", res.GetLicense())
	fmt.Printf("Starts: %d\n", res.GetStarsCount())
	fmt.Printf("Forks: %d\n", res.GetForksCount())
	fmt.Printf("Issues: %d\n", res.GetOpenIssuesCount())

	fmt.Printf("Scorecard\n")
	scorecard := res.GetScorecard()

	fmt.Printf("  Score: %.2f\n", scorecard.GetOverallScore())
	fmt.Printf("  Checks\n")

	checks := scorecard.GetChecks()
	for _, check := range checks {
		fmt.Printf("    %s: %d\n", check.GetName(), check.GetScore())
		fmt.Printf("      %s\n", check.GetReason())
	}
}
