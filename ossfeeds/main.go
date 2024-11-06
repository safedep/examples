package main

import (
	"fmt"
	"time"

	"github.com/ossf/package-feeds/pkg/events"
	"github.com/ossf/package-feeds/pkg/feeds"
	"github.com/ossf/package-feeds/pkg/feeds/npm"
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
		}

		cutoff = newCutoff
	}
}
