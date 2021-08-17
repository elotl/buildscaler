package scraper

import (
	"context"
	"k8s.io/klog/v2"
	"sync"
)

type FakeScraper struct {
	storage *sync.Map
}

func New(storage *sync.Map) *FakeScraper {
	return &FakeScraper{storage: storage}
}

func (f *FakeScraper) Scrape(cancel context.CancelFunc) error {
	klog.V(5).Info("scraping metrics from FakeProvider...")
	if false {
		cancel()
	}
	return nil
}

func (f *FakeScraper) GetMetricValue(name string) interface{} {
	v, _ := f.storage.Load(name)
	return v
}
