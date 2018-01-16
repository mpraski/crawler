package main

import (
	"sync"
)

// Options struct represents list of optional parameters to the Crawler.
// MaxWorker defines the number of goroutines spawned to process the downloaded websites,
// MaxRetries defined how many times should the crawler try to reach any website,
// Downloader and Extractor are two depencies on which the Crawler relies,
// Callback is a reference to the function called upon discovering new URL.
type Options struct {
	MaxWorkers, MaxRetries int
	Downloader             Downloader
	Extractor              Extractor
	Callback               func(string)
}

var defaultOptions = Options{
	MaxWorkers: 10,
	MaxRetries: 2,
}

// Crawler struct represents the web crawler which takes the root url, a list of parameters and produces a sitemap.
type Crawler struct {
	// Root URL
	url string

	maxRetries, maxWorkers int

	downloader Downloader
	extractor  Extractor

	// Waitgroups for controlling termination of the main program and the goroutines
	wg, wgStop sync.WaitGroup

	// Since this map can be accessed by multiple goroutines, it is guarded with a mutex
	mus   sync.RWMutex
	sites map[string]*Page

	// Since this map can be accessed by multiple goroutines, it is guarded with a mutex
	mur     sync.RWMutex
	retries map[string]int

	// Since this map can be accessed by multiple goroutines, it is guarded with a mutex
	mup       sync.RWMutex
	processed map[string]bool

	// internal channels for communicating crawler results and terminating workers
	results chan *result
	quit    []chan struct{}

	// external channels for signalling crawler termination and errors
	done   chan struct{}
	errors chan error

	callback func(string)
}

func NewCrawler(url string) (*Crawler, error) {
	c := &Crawler{
		url: url,

		maxRetries: defaultOptions.MaxRetries,
		maxWorkers: defaultOptions.MaxWorkers,

		downloader: NewDefaultDownloader(2, NewBufferPool(10, 1024)),

		results: make(chan *result, defaultOptions.MaxWorkers),
		quit:    make([]chan struct{}, 0, defaultOptions.MaxWorkers),

		done:   make(chan struct{}),
		errors: make(chan error, 100),

		sites:     make(map[string]*Page),
		retries:   make(map[string]int),
		processed: make(map[string]bool),
	}

	if extractor, err := NewDefaultExtractor(url); err == nil {
		c.extractor = extractor
	} else {
		return nil, err
	}

	return c, nil
}

func NewCrawlerWithOptions(url string, options *Options) (*Crawler, error) {
	c := &Crawler{
		url: url,

		maxRetries: options.MaxRetries,
		maxWorkers: options.MaxWorkers,

		results: make(chan *result, options.MaxWorkers),
		quit:    make([]chan struct{}, 0, options.MaxWorkers),

		done:   make(chan struct{}),
		errors: make(chan error, 100),

		sites:     make(map[string]*Page),
		retries:   make(map[string]int),
		processed: make(map[string]bool),
	}

	if options.Downloader != nil {
		c.downloader = options.Downloader
	} else {
		c.downloader = NewDefaultDownloader(2, NewBufferPool(10, 1024))
	}

	if options.Extractor != nil {
		c.extractor = options.Extractor
	} else {
		if ext, err := NewDefaultExtractor(url); err == nil {
			c.extractor = ext
		} else {
			return nil, err
		}
	}

	if options.Callback != nil {
		c.callback = options.Callback
	}

	return c, nil
}

func (c *Crawler) Crawl() (chan struct{}, chan error) {
	c.wgStop.Add(c.maxWorkers)

	for i := 0; i < c.maxWorkers; i++ {
		q := make(chan struct{})
		c.quit = append(c.quit, q)

		go c.collect(q)
	}

	go func() {
		c.wg.Add(1)

		c.sites["<root>"] = &Page{
			LinkedFrom: make([]*Page, 0),
			LinksTo:    make([]*Page, 0),
			Assets:     make([]*Asset, 0),
		}

		c.crawl(c.url, "<root>")
		c.wg.Wait()

		c.stopGoroutines()
		c.wgStop.Wait()

		close(c.results)
		delete(c.sites, "<root>")

		c.retries = nil
		c.processed = nil

		c.done <- struct{}{}
	}()

	return c.done, c.errors
}

func (c *Crawler) GetSiteMap() map[string]*Page {
	return c.sites
}

func (c *Crawler) stopGoroutines() {
	for i, _ := range c.quit {
		c.quit[i] <- struct{}{}
	}
}

func (c *Crawler) crawl(url, from string) {
	var (
		body []byte
		err  error
	)

	if body, err = c.downloader.Download(url); err == nil {
		c.markBeingProcessed(url, false)

		c.results <- &result{
			url:  url,
			from: from,
			body: body,
		}
	} else {
		c.errors <- err

		if c.shouldRetry(url) {
			c.markRetry(url)

			c.crawl(url, from)
		} else {
			c.markBeingProcessed(url, false)

			c.wg.Done()
		}
	}
}

func (c *Crawler) collect(quit <-chan struct{}) {
	defer c.wgStop.Done()

	for {
		select {
		case result := <-c.results:
			var (
				title  string
				links  []string
				assets []*Asset
				err    error
			)

			if title, links, assets, err = c.extractor.Extract(result.body); err == nil {
				page := &Page{
					Title:      title,
					Url:        result.url,
					LinkedFrom: make([]*Page, 0),
					LinksTo:    make([]*Page, 0),
					Assets:     assets,
				}

				c.markVisited(result.url, page)
				c.addLinksTo(result.from, page)

				for _, link := range links {
					if c.hasVisited(link) {
						c.addLinkedFrom(link, page)
					} else {
						if !c.isBeingProcessed(link) && c.shouldRetry(link) {
							c.markBeingProcessed(link, true)

							c.wg.Add(1)

							go func(url, from string) {
								c.crawl(url, from)
							}(link, result.url)

							if c.callback != nil {
								go func(s string) {
									c.callback(s)
								}(link)
							}
						}
					}
				}
			} else {
				c.errors <- err
			}

			c.wg.Done()
		case <-quit:
			return
		}
	}
}

func (c *Crawler) hasVisited(url string) bool {
	c.mus.RLock()
	var _, ok = c.sites[url]
	c.mus.RUnlock()

	return ok
}

func (c *Crawler) markVisited(url string, page *Page) {
	c.mus.Lock()
	c.sites[url] = page
	c.mus.Unlock()
}

func (c *Crawler) addLinkedFrom(url string, page *Page) {
	c.mus.Lock()
	c.sites[url].LinkedFrom = append(c.sites[url].LinkedFrom, page)
	c.mus.Unlock()
}

func (c *Crawler) addLinksTo(url string, page *Page) {
	c.mus.Lock()
	c.sites[url].LinksTo = append(c.sites[url].LinksTo, page)
	c.mus.Unlock()
}

func (c *Crawler) isBeingProcessed(url string) bool {
	c.mup.RLock()
	value := c.processed[url]
	c.mup.RUnlock()

	return value
}

func (c *Crawler) markBeingProcessed(url string, processed bool) {
	c.mup.Lock()
	c.processed[url] = processed
	c.mup.Unlock()
}

func (c *Crawler) shouldRetry(url string) bool {
	c.mur.RLock()
	value := c.retries[url]
	c.mur.RUnlock()

	return value < c.maxRetries
}

func (c *Crawler) markRetry(url string) {
	c.mur.Lock()
	c.retries[url] = c.retries[url] + 1
	c.mur.Unlock()
}
