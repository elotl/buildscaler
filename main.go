package main

import (
	"context"
	"flag"
	"github.com/elotl/ciplatforms-external-metrics/pkg/ciprovider"
	"github.com/elotl/ciplatforms-external-metrics/pkg/scraper"
	"k8s.io/component-base/logs"
	"k8s.io/klog/v2"
	"os"
	"sigs.k8s.io/custom-metrics-apiserver/pkg/cmd"
	"sigs.k8s.io/custom-metrics-apiserver/pkg/provider"
	"sync"
	"time"
)

type YourAdapter struct {
	cmd.AdapterBase
	// the message printed on startup
	Message string
}

func (a *YourAdapter) makeProviderOrDie(storage *sync.Map) provider.ExternalMetricsProvider {
	//client, err := a.DynamicClient()
	//if err != nil {
	//	klog.Fatalf("unable to construct dynamic client: %v", err)
	//}
	//
	//mapper, err := a.RESTMapper()
	//if err != nil {
	//	klog.Fatalf("unable to construct discovery REST mapper: %v", err)
	//}
	return ciprovider.NewFakeProvider(storage)
}

func main() {
	adapter := &YourAdapter{Message: "buildkite_adapter"}
	logs.InitLogs()
	defer logs.FlushLogs()
	var scrapePeriod time.Duration
	adapter.Flags().StringVar(&adapter.Message, "msg", "starting adapter...", "startup message")
	adapter.Flags().DurationVar(&scrapePeriod, "scrape-period", time.Second*5, "scrape period")
	adapter.Flags().AddGoFlagSet(flag.CommandLine) // make sure you get the klog flags
	err := adapter.Flags().Parse(os.Args)
	if err != nil {
		klog.Fatal(err)
	}
	storage := &sync.Map{}

	externalMetricsProvider := adapter.makeProviderOrDie(storage)
	adapter.WithExternalMetrics(externalMetricsProvider)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	klog.Infof(adapter.Message)
	fakeScraper := scraper.New(storage)
	ticker := time.NewTicker(scrapePeriod)
	if err := adapter.Run(ctx.Done()); err != nil {
		klog.Fatalf("unable to run metrics adapter: %v", err)
	}
	for {
		go func() {
			err := fakeScraper.Scrape(cancel)
			if err != nil {
				klog.Error(err)
			}
		}()
		select {
		case <-ctx.Done():
			klog.V(3).Info("scraping finished.")
			return
		case <-ticker.C:
		}
	}

}
