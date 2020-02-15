package domain

import (
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/sendoushi/scrapper/config"
)

// FetchURL will fetch an URL
func FetchURL(url string, c config.Config) ([]map[string]string, error) {
	var pagesVisited []string
	var dataArr []map[string]string

	limit := colly.LimitRule{
		// DomainGlob:  "*httpbin.*",
		Parallelism: 2,
		RandomDelay: 5 * time.Second,
	}

	collector := colly.NewCollector(
		// colly.AllowedDomains("racius.com"),

		// Cache responses to prevent multiple download of pages
		// even if the collector is restarted
		colly.CacheDir(c.GetCacheFolder()),
	)
	collector.Limit(&limit)

	// Create another collector to scrape company details
	detailCollector := collector.Clone()

	collector.OnRequest(func(r *colly.Request) {
		c.GetLogger().Log("-- visiting", r.URL.String())
	})

	collector.OnHTML(c.GetDataURLSelector(), func(e *colly.HTMLElement) {
		// don't request more than our limit
		if c.GetMaxPageRequests() < len(pagesVisited) {
			return
		}

		page := e.Request.AbsoluteURL(e.Attr("href"))

		// dont go further if we have already gone there
		for _, pageVisited := range pagesVisited {
			if page == pageVisited {
				return
			}
		}

		// dont go further if we have the single in already
		for _, single := range dataArr {
			if single["URL"] == page {
				return
			}
		}

		pagesVisited = append(pagesVisited, page)

		c.GetLogger().Log("--- visiting_detail", page)
		detailCollector.Visit(page)
	})

	// Extract details of the single
	detailCollector.OnHTML(c.GetDataSelector(), func(e *colly.HTMLElement) {
		page := e.Request.URL.String()

		// dont go further if we have the single in already
		for _, single := range dataArr {
			if single["URL"] == page {
				return
			}
		}

		single, err := getSingleDetails(e, c)
		if err != nil {
			// DEV: it must be something related to a required, ignore
			return
		}

		// Save the single
		single["URL"] = page
		dataArr = append(dataArr, single)
	})

	collector.OnHTML(c.GetCrawlSelector(), func(e *colly.HTMLElement) {
		// don't request more than our limit
		if c.GetMaxPageRequests() < len(pagesVisited) {
			return
		}

		page := e.Request.AbsoluteURL(e.Attr("href"))

		// dont go further if we have already gone there
		for _, pageVisited := range pagesVisited {
			if page == pageVisited {
				return
			}
		}

		pagesVisited = append(pagesVisited, page)
		collector.Visit(page)
	})

	// Start scraping
	collector.Visit(c.GetBaseURL())

	// Wait until threads are finished
	collector.Wait()
	detailCollector.Wait()

	return dataArr, nil
}