package main

import (
	"flag"
	"fmt"
)

func main() {
	var (
		argAddress = flag.String("address", "", "The address to be crawled")
		argWorkers = flag.Int("workers", 10, "Number of workers processing the crawled websites")
		argRetries = flag.Int("retries", 2, "Number of retries for each website")
	)

	flag.Parse()

	fmt.Printf("Params: (Address: %s), (Workers: %d), (Retries: %d)\n\n", *argAddress, *argWorkers, *argRetries)

	crawler, err := NewCrawlerWithOptions(*argAddress, &Options{
		MaxWorkers: *argWorkers,
		MaxRetries: *argRetries,
		Callback: func(s string) {
			fmt.Printf("Crawling: %s\n", s)
		},
	})

	if err != nil {
		panic(err)
	}

	done, errors := crawler.Crawl()

	go func() {
		select {
		case e := <-errors:
			fmt.Printf("Error: %s\n", e.Error())
		}
	}()

	<-done

	fmt.Printf("\n\033[1mResults:\033[0m\n\n")

	for k, v := range crawler.GetSiteMap() {
		fmt.Printf("─────────────────────────────────────────────────\n")
		fmt.Printf("Crawled \033[1m%s\033[0m | %s\n", k, v.Title)
		fmt.Printf(" ╠ \033[1mAssets:\033[0m\n")
		for _, asset := range v.Assets {
			fmt.Printf(" ╠══ %s\n", asset.Url)
		}
		if len(v.LinksTo) > 0 {
			fmt.Printf(" ╠ \033[1mLinks to:\033[0m\n")
			for _, page := range v.LinksTo {
				fmt.Printf(" ╠══ %s\n", page.Url)
			}
		}
		if len(v.LinkedFrom) > 0 {
			fmt.Printf(" ╠ \033[1mLinked from:\033[0m\n")
			for _, page := range v.LinkedFrom {
				fmt.Printf(" ╠══ %s\n", page.Url)
			}
		}
		fmt.Printf("─────────────────────────────────────────────────\n\n")
	}
}
