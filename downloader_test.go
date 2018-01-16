package main

import (
	"testing"
)

func TestDownloaderFetchesCorrectly(t *testing.T) {
	const (
		ADDRESS      = "http://marcinpraski.com/"
		FETCHED_SIZE = 22671
	)

	var (
		timeout    = 5
		bp         = NewBufferPool(2, 1024)
		downloader = NewDefaultDownloader(timeout, bp)
		data       []byte
		err        error
	)

	if data, err = downloader.Download(ADDRESS); err != nil {
		t.Errorf("Downloader fails with error: %s\n", err.Error())
	}

	if len(data) != FETCHED_SIZE {
		t.Errorf("Size of downloaded data mismatch: %d\n", len(data))
	}
}
