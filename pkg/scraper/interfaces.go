package scraper

import "context"

type CIScraper interface {
	Scrape(cancel context.CancelFunc) error
	GetMetricValue(name string) interface{}
}
