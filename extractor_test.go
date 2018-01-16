package main

import (
	"testing"
)

func TestExtractorAcceptsOnlyValidURLs(t *testing.T) {
	var (
		valid = []string{
			"https://www.google.co.uk",
			"https://monzo.me/",
			"https://bernsteinbear.com/blog/lisp/",
			"https://gcc.gnu.org/onlinedocs/cpp/Concatenation.html",
		}

		invalid = []string{
			"/some/path",
			"monzo.me/",
			"https//bernsteinbear.com/blog/lisp/",
			"gcc.gnu.org/onlinedocs/cpp/Concatenation.html",
		}
	)

	for _, v := range valid {
		if _, err := NewDefaultExtractor(v); err != nil {
			t.Errorf("Extractor fails for valid URL: %s\n", v)
		}
	}

	for _, v := range invalid {
		if _, err := NewDefaultExtractor(v); err == nil {
			t.Errorf("Extractor does not fail for invalid URL: %s\n", v)
		}
	}
}

func TestExtractorRecongizesTags(t *testing.T) {
	var (
		url = "http://example.com/"

		html = `
		<html class="no-js" lang="">
		    <head>
		        <meta charset="utf-8">
		        <meta http-equiv="x-ua-compatible" content="ie=edge">
		        <title>The Title</title>
		        <meta name="description" content="">
		        <meta name="viewport" content="width=device-width, initial-scale=1, shrink-to-fit=no">

		        <link rel="manifest" href="site.webmanifest">
		        <link rel="apple-touch-icon" href="icon.png">

		        <link rel="stylesheet" href="css/normalize.css">
		        <link rel="stylesheet" href="css/main.css">
		    </head>
		    <body>
		        <p>Hello world! This is HTML5 Boilerplate.</p>
		        <a href="/some-other-page">Click me</a>
		        <a href="/some-other-page2.html">Click me</a>
		        <script src="js/vendor/modernizr-1.0.min.js"></script>
		        <script>window.jQuery || document.write('<script src="js/vendor/jquery-{{JQUERY_VERSION}}.min.js"><\/script>')</script>
		        <script src="js/plugins.js"></script>
		        <script src="js/main.js"></script>
		    </body>
		</html>
		`

		expectedTitle = "The Title"

		expectedAssets = []*Asset{
			&Asset{
				Url:  "http://example.com/site.webmanifest",
				Type: Link,
			},
			&Asset{
				Url:  "http://example.com/icon.png",
				Type: Link,
			},
			&Asset{
				Url:  "http://example.com/css/normalize.css",
				Type: Link,
			},
			&Asset{
				Url:  "http://example.com/css/main.css",
				Type: Link,
			},
			&Asset{
				Url:  "http://example.com/js/vendor/modernizr-1.0.min.js",
				Type: Script,
			},
			&Asset{
				Url:  "http://example.com/js/plugins.js",
				Type: Script,
			},
			&Asset{
				Url:  "http://example.com/js/main.js",
				Type: Script,
			},
		}

		expectedLinks = []string{
			"http://example.com/some-other-page",
			"http://example.com/some-other-page2.html",
		}
	)

	var (
		e   Extractor
		err error
	)

	if e, err = NewDefaultExtractor(url); err != nil {
		t.Errorf("Extractor fails for URL %s with error: %s\n", url, err.Error())
	}

	var (
		title  string
		links  []string
		assets []*Asset
	)

	if title, links, assets, err = e.Extract([]byte(html)); err != nil {
		t.Errorf("Extractor fails with error: %s\n", err.Error())
	}

	if title != expectedTitle {
		t.Errorf("Unexpected title: %s\n", title)
	}

	for i := range assets {
		if assets[i].Url != expectedAssets[i].Url || assets[i].Type != expectedAssets[i].Type {
			t.Errorf("Unexpected asset: %s\n", assets[i].Url)
		}
	}

	for i := range links {
		if links[i] != expectedLinks[i] {
			t.Errorf("Unexpected link: %s\n", links[i])
		}
	}
}
