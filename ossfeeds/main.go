package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"buf.build/gen/go/safedep/api/grpc/go/safedep/services/malysis/v1/malysisv1grpc"
	malysisv1pb "buf.build/gen/go/safedep/api/protocolbuffers/go/safedep/messages/malysis/v1"
	packagev1 "buf.build/gen/go/safedep/api/protocolbuffers/go/safedep/messages/package/v1"
	malysisv1 "buf.build/gen/go/safedep/api/protocolbuffers/go/safedep/services/malysis/v1"
	"github.com/ossf/package-feeds/pkg/events"
	"github.com/ossf/package-feeds/pkg/feeds"
	"github.com/ossf/package-feeds/pkg/feeds/npm"
	drygrpc "github.com/safedep/dry/adapters/grpc"
	"google.golang.org/grpc"
)

type npmEventHandler struct{}

func (n *npmEventHandler) AddEvent(e events.Event) error {
	fmt.Printf("Event: Type: %s Message: %s\n",
		e.GetType(), e.GetMessage())

	return nil
}

func main() {
	npmFeed, err := npm.New(feeds.FeedOptions{}, events.NewHandler(&npmEventHandler{}, events.Filter{
		EnabledEventTypes: []string{events.LossyFeedEventType, events.FeedsComponentType},
	}))
	if err != nil {
		panic(err)
	}

	tok := os.Getenv("SAFEDEP_API_KEY")
	tenantId := os.Getenv("SAFEDEP_TENANT_ID")

	if tok == "" || tenantId == "" {
		panic("SAFEDEP_API_KEY and SAFEDEP_TENANT_ID must be set")
	}

	headers := http.Header{}
	headers.Set("x-tenant-id", tenantId)

	cc, err := drygrpc.GrpcClient("pkg-feed-client", "api.safedep.io", "443",
		tok, headers, []grpc.DialOption{})
	if err != nil {
		panic(err)
	}

	service := malysisv1grpc.NewMalwareAnalysisServiceClient(cc)

	cutoff := time.Now().Add(-time.Hour * 1)
	for {
		packages, newCutoff, errs := npmFeed.Latest(cutoff)
		if len(errs) > 0 {
			fmt.Printf("Error polling feed: %v\n", errs)
			continue
		}

		for _, pkg := range packages {
			fmt.Printf("Type: %s Package: %s, Version: %s SchemaVer: %s\n",
				pkg.Type, pkg.Name, pkg.Version, pkg.SchemaVer)

			if err := submitForAnalysis(service, pkg); err != nil {
				fmt.Printf("Error submitting package for analysis: %v\n", err)
			}
		}

		cutoff = newCutoff
	}
}

func submitForAnalysis(client malysisv1grpc.MalwareAnalysisServiceClient, pkg *feeds.Package) error {
	req := malysisv1.AnalyzePackageRequest{
		Target: &malysisv1pb.PackageAnalysisTarget{
			PackageVersion: &packagev1.PackageVersion{
				Package: &packagev1.Package{
					Ecosystem: packagev1.Ecosystem_ECOSYSTEM_NPM,
					Name:      pkg.Name,
				},
				Version: pkg.Version,
			},
		},
	}

	res, err := client.AnalyzePackage(context.Background(), &req)
	if err != nil {
		return fmt.Errorf("error submitting package for analysis: %w", err)
	}

	fmt.Printf("Submitted package for analysis: %s\n", res.GetAnalysisId())

	return nil
}
