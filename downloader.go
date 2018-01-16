package main

import (
	"net/http"
	"time"
)

// Downloader interface abstracts the operation of fetching the website's content under given URL.
type Downloader interface {
	Download(url string) (body []byte, err error)
}

// defaultDownloader implementation uses a http.Client with user defined timeout to fetch the content.
// To save memory between subsequent calls the response is read to a buffer taken from the buffer pool
// whose parameters (initial number of buffers and size of each) are specified by the caller
type defaultDownloader struct {
	client *http.Client
	pool   *BufferPool
}

func NewDefaultDownloader(timeout int, pool *BufferPool) Downloader {
	return &defaultDownloader{
		client: &http.Client{
			Timeout: time.Second * time.Duration(timeout),
		},
		pool: pool,
	}
}

func (d *defaultDownloader) Download(url string) ([]byte, error) {
	var (
		resp *http.Response
		err  error
	)

	resp, err = d.client.Get(url)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, ErrBadResponse
	}

	defer resp.Body.Close()

	b := d.pool.Get()
	defer d.pool.Put(b)

	if _, err = b.ReadFrom(resp.Body); err == nil {
		return b.Bytes(), nil
	} else {
		return nil, err
	}
}
