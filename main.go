/*
Copyright 2022 Elotl Inc.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/elotl/buildscaler/pkg/ciprovider"
	"github.com/elotl/buildscaler/pkg/collector"
	storagemap "github.com/elotl/buildscaler/pkg/storage"

	"k8s.io/component-base/logs"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	"sigs.k8s.io/custom-metrics-apiserver/pkg/cmd"
)

var (
	BuildkitePlatform  = "buildkite"
	CircleCIPlatform   = "circleci"
	FlarebuildPlatform = "flarebuild"

	CIPlatforms = []string{
		BuildkitePlatform,
		CircleCIPlatform,
		FlarebuildPlatform,
	}
)

func createMetricCollector(ciPlatform string, storage *storagemap.ExternalMetricsMap) (collector.CIMetricsCollector, error) {
	switch ciPlatform {
	case CircleCIPlatform:
		// TODO
		token, projectSlug := GetCircleCIConfigFromEnvOrDie()
		metricsCollector, err := collector.NewCircleCICollector(token, projectSlug, time.Minute*30, storage)
		if err != nil {
			klog.Errorf("cannot start CircleCI scraper: %s", err)
			return nil, err
		}
		return metricsCollector, nil
	case BuildkitePlatform:
		token := GetBuildkiteTokenFromEnvOrDie()
		queues := GetBuildkiteQueuesFromEnv()
		return collector.NewBuildkiteCollector(storage, token, "v0.0.1", queues), nil
	case FlarebuildPlatform:
		var apiKey, endpoint = GetFlarebuildConfigFromEnvOrDie()
		return collector.NewFlarebuild(storage, apiKey, endpoint)
	default:
		return nil, fmt.Errorf("unknown ci platform: %s", ciPlatform)
	}
}

func main() {
	adapter := &cmd.AdapterBase{
		Name: "buildscaler",
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
	storage := storagemap.NewExternalMetricsMap()
	metricsCollector, err := createMetricCollector(CIPlatform, storage)
	if err != nil {
		klog.Fatal(err)
	}

	klog.V(2).Infof("using %s scraper & metrics provider", CIPlatform)
	externalMetricsProvider := ciprovider.NewExternalMetricsProviderFromStorage(storage)
	adapter.WithExternalMetrics(externalMetricsProvider)

	ctx, cancel := context.WithCancel(signals.SetupSignalHandler())
	defer cancel()

	var serverDone = make(chan struct{})
	go func() {
		if err := adapter.Run(ctx.Done()); err != nil {
			cancel()
			klog.Fatalf("unable to run metrics adapter: %v", err)
		}
		close(serverDone)
	}()

	ticker := time.NewTicker(scrapePeriod)
	for {
		err := metricsCollector.Collect(cancel)
		if err != nil {
			klog.Errorf("error scraping metrics: %s", err)
		}
		select {
		case <-ctx.Done():
			klog.Info("Finished.")
			<-serverDone // Wait for metrics adapter to finish
			return
		case <-ticker.C:
		}
	}
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

func GetFlarebuildConfigFromEnvOrDie() (string, string) {
	var apiKey = os.Getenv("FLAREBUILD_API_KEY")
	if apiKey == "" {
		klog.Fatal("environment variable FLAREBUILD_API_KEY not set")
	}
	var endpoint = os.Getenv("FLAREBUILD_ENDPOINT")
	if endpoint == "" {
		endpoint = "https://api.stg.flare.build/api/v1"
	}
	klog.V(2).Infof("using %s as endpoint", endpoint)
	return apiKey, endpoint
}
