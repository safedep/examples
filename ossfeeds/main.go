package main

import (
	"context"
	"flag"
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
	"github.com/ossf/package-feeds/pkg/feeds/pypi"
	"github.com/ossf/package-feeds/pkg/feeds/rubygems"
	drygrpc "github.com/safedep/dry/adapters/grpc"
	"google.golang.org/grpc"
)

const (
	npmEcosystem      = "npm"
	rubygemsEcosystem = "rubygems"
	pypiEcosystem     = "pypi"
)

type eventHandler struct{}

func (n *eventHandler) AddEvent(e events.Event) error {
	fmt.Printf("Event: Type: %s Message: %s\n",
		e.GetType(), e.GetMessage())

	return nil
}

type feedListener interface {
	Latest(cutoff time.Time) ([]*feeds.Package, time.Time, []error)
}

func buildFeedListener(name string) (feedListener, error) {
	var feedListener feedListener
	var err error

	switch name {
	case npmEcosystem:
		feedListener, err = npm.New(feeds.FeedOptions{}, events.NewHandler(&eventHandler{},
			events.Filter{
				EnabledEventTypes: []string{events.LossyFeedEventType, events.FeedsComponentType},
			}))
	case rubygemsEcosystem:
		feedListener, err = rubygems.New(feeds.FeedOptions{}, events.NewHandler(&eventHandler{},
			events.Filter{
				EnabledEventTypes: []string{events.LossyFeedEventType, events.FeedsComponentType},
			}))
	case pypiEcosystem:
		feedListener, err = pypi.New(feeds.FeedOptions{}, events.NewHandler(&eventHandler{},
			events.Filter{
				EnabledEventTypes: []string{events.LossyFeedEventType, events.FeedsComponentType},
			}))
	default:
		err = fmt.Errorf("unsupported feed: %s", name)
	}

	return feedListener, err
}

var inputEcosystem string

func init() {
	flag.StringVar(&inputEcosystem, "ecosystem", npmEcosystem, "Ecosystem to poll")
	flag.Parse()
}

func main() {
	feedListener, err := buildFeedListener(inputEcosystem)
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
		packages, newCutoff, errs := feedListener.Latest(cutoff)
		if len(errs) > 0 {
			fmt.Printf("Error polling feed: %v\n", errs)
			continue
		}

		for _, pkg := range packages {
			fmt.Printf("Type: %s Package: %s, Version: %s SchemaVer: %s\n",
				pkg.Type, pkg.Name, pkg.Version, pkg.SchemaVer)

			if err := submitForAnalysis(service, inputEcosystem, pkg); err != nil {
				fmt.Printf("Error submitting package for analysis: %v\n", err)
			}
		}

		cutoff = newCutoff
	}
}

func submitForAnalysis(client malysisv1grpc.MalwareAnalysisServiceClient, ecosystem string, pkg *feeds.Package) error {
	specEcosystem := packagev1.Ecosystem_ECOSYSTEM_UNSPECIFIED
	switch ecosystem {
	case npmEcosystem:
		specEcosystem = packagev1.Ecosystem_ECOSYSTEM_NPM
	case rubygemsEcosystem:
		specEcosystem = packagev1.Ecosystem_ECOSYSTEM_RUBYGEMS
	case pypiEcosystem:
		specEcosystem = packagev1.Ecosystem_ECOSYSTEM_PYPI
	}

	req := malysisv1.AnalyzePackageRequest{
		Target: &malysisv1pb.PackageAnalysisTarget{
			PackageVersion: &packagev1.PackageVersion{
				Package: &packagev1.Package{
					Ecosystem: specEcosystem,
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
