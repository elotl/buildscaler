package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/elotl/buildscaler/pkg/ciprovider"
	"github.com/elotl/buildscaler/pkg/collector"
	storagemap "github.com/elotl/buildscaler/pkg/storage"
	"k8s.io/component-base/logs"
	"k8s.io/klog/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"
	"os"
	"sigs.k8s.io/custom-metrics-apiserver/pkg/cmd"
	"strings"
	"sync"
	"time"
)

var (
	BuildkitePlatform = "buildkite"
	CircleCIPlatform  = "circleci"

	CIPlatforms = []string{
		BuildkitePlatform,
		CircleCIPlatform,
	}
)

func main() {
	adapter := &cmd.AdapterBase{
		Name: "ci-platforms-metrics-adapter",
	}
	logs.InitLogs()
	defer logs.FlushLogs()
	var scrapePeriod time.Duration
	var CIPlatform string
	adapter.Flags().DurationVar(&scrapePeriod, "scrape-period", time.Second*5, "scrape period")
	adapter.Flags().StringVar(
		&CIPlatform,
		"ci-platform",
		BuildkitePlatform,
		fmt.Sprintf("specify CI platform for scraping metrics. \nSupported platforms: %s", CIPlatforms),
	)
	adapter.Flags().AddGoFlagSet(flag.CommandLine) // make sure you get the klog flags
	err := adapter.Flags().Parse(os.Args)
	if err != nil {
		klog.Fatal(err)
	}
	rwm := &sync.RWMutex{}
	storage := &storagemap.ExternalMetricsMap{
		RWMutex: rwm,
		Data:    make(map[string]external_metrics.ExternalMetricValue),
	}
	var metricsCollector collector.CIMetricsCollector
	switch CIPlatform {
	case CircleCIPlatform:
		// TODO
		token, projectSlug := GetCircleCIConfigFromEnvOrDie()
		metricsCollector, err = collector.NewCircleCICollector(token, projectSlug, time.Minute*30, storage)
		if err != nil {
			klog.Fatalf("cannot start CircleCI scraper: %v", err)
		}
	case BuildkitePlatform:
		token := GetBuildkiteTokenFromEnvOrDie()
		queues := GetBuildkiteQueuesFromEnv()
		metricsCollector = collector.NewBuildkiteCollector(storage, token, "v0.0.1", queues)
	default:
		klog.Fatal("unknown ci platform")
	}
	klog.V(2).Infof("using %s scraper & metrics provider", CIPlatform)
	externalMetricsProvider := ciprovider.NewExternalMetricsProviderFromStorage(storage)
	adapter.WithExternalMetrics(externalMetricsProvider)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ticker := time.NewTicker(scrapePeriod)
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		if err := adapter.Run(ctx.Done()); err != nil {
			klog.Fatalf("unable to run metrics adapter: %v", err)
			wg.Done()
			cancel()
		}

	}()
	klog.V(2).Infof("server runs, let's start a scraper...")
	wg.Add(1)
	go func() {
		for {
			err := metricsCollector.Collect(cancel)
			if err != nil {
				klog.Errorf("error scraping metrics: %v", err)
				cancel()
			}
			select {
			case <-ctx.Done():
				klog.V(3).Info("scraping finished.")
				wg.Done()
				return
			case <-ticker.C:
			}
		}
	}()
	wg.Wait()

}

func GetBuildkiteTokenFromEnvOrDie() string {
	token := os.Getenv("BUILDKITE_AGENT_TOKEN")
	if token == "" {
		klog.Fatal("cannot get Buildkite Agent Token from BUILDKITE_AGENT_TOKEN env var")
	}
	return token
}

func GetBuildkiteQueuesFromEnv() []string {
	queuesStr := os.Getenv("BUILDKITE_QUEUES")
	if queuesStr == "" {
		return []string{}
	}
	queues := strings.Split(queuesStr, ",")
	return queues
}

func GetCircleCIConfigFromEnvOrDie() (string, string) {
	token := os.Getenv("CIRCLECI_TOKEN")
	if token == "" {
		klog.Fatal("cannot get CircleCI API Token from CIRCLECI_TOKEN env var")
	}
	projectSlug := os.Getenv("CIRCLECI_PROJECT_SLUG")
	if projectSlug == "" {
		klog.Fatalf("cannot get CircleCI project slug from CIRCLECI_PROJECT_SLUG en var")
	}
	return token, projectSlug
}
