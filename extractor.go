package main

import (
	"bytes"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

// Extractor interface abstract the operation of extracting interesting pieces of data from the content.
// As of now the website's title, list of internal hyperlings and list of static assets are extracted
// If extraction fails an error is returned.
type Extractor interface {
	Extract(body []byte) (name string, links []string, assets []*Asset, err error)
}

// defaultExtractor implementation uses the golang.org/x/net/html for tokenizing the html tree.
// A unique list of links and assets is generated.
type defaultExtractor struct {
	domain    *url.URL
	fileRegex *regexp.Regexp
}

func NewDefaultExtractor(domain string) (Extractor, error) {
	u, err := url.ParseRequestURI(domain)
	if err != nil {
		return nil, err
	}

	if u.Scheme == "" || u.Host == "" {
		return nil, ErrInvalidURL
	}

	r := regexp.MustCompile("^(/.*){0,}[\\w,\\s-]+\\.[A-Za-z]{1,}$")

	return &defaultExtractor{
		domain:    u,
		fileRegex: r,
	}, nil
}

func (d *defaultExtractor) Extract(body []byte) (string, []string, []*Asset, error) {
	var (
		z                   *html.Tokenizer     = html.NewTokenizer(bytes.NewReader(body))
		setLinks, setAssets map[string]struct{} = make(map[string]struct{}), make(map[string]struct{})
		title               string
		links               []string = make([]string, 0)
		assets              []*Asset = make([]*Asset, 0)
	)

	for {
		tt := z.Next()

		if tt == html.ErrorToken {
			if z.Err() == io.EOF {
				break
			}

			return "", []string{}, []*Asset{}, z.Err()
		}

		if tt == html.StartTagToken {
			t := z.Token()
			switch t.Data {
			case "title":
				tt := z.Next()

				if tt == html.TextToken {
					title = strings.TrimSpace(z.Token().Data)
				}
			case "a":
				for _, a := range t.Attr {
					if a.Key == "href" {
						if d.isFileUrl(a.Val) {
							expanded := d.expandIfNeeded(a.Val)
							if _, ok := setAssets[expanded]; !ok {
								assets = append(assets, &Asset{Url: expanded, Type: Link})
								setAssets[expanded] = struct{}{}
							}
						} else if d.isSameDomain(a.Val) {
							expanded := d.expandIfNeeded(a.Val)
							if _, ok := setLinks[expanded]; !ok {
								links = append(links, expanded)
								setLinks[expanded] = struct{}{}
							}
						}
					}
				}
			case "script":
				for _, a := range t.Attr {
					if a.Key == "src" {
						d.addAsset(&assets, setAssets, a.Val, Script)
					}
				}
			case "img":
				for _, a := range t.Attr {
					if a.Key == "src" {
						d.addAsset(&assets, setAssets, a.Val, Image)
					}
				}
			case "link":
				for _, a := range t.Attr {
					if a.Key == "href" {
						d.addAsset(&assets, setAssets, a.Val, Link)
					}
				}
			case "source":
				if tt == html.TextToken {
					d.addAsset(&assets, setAssets, z.Token().Data, Video)
				}
			}
		}
	}

	return title, links, assets, nil
}

func (d *defaultExtractor) addAsset(assets *[]*Asset, set map[string]struct{}, address string, kind AssetType) {
	if d.isFileUrl(address) {
		expanded := d.expandIfNeeded(address)
		if _, ok := set[expanded]; !ok {
			*assets = append(*assets, &Asset{Url: expanded, Type: kind})
			set[expanded] = struct{}{}
		}
	}
}

func (d *defaultExtractor) isSameDomain(address string) bool {
	u, err := url.Parse(address)
	if err != nil {
		return false
	}

	return (u.Host == "") || d.domain.Host == u.Host
}

func (d *defaultExtractor) expandIfNeeded(address string) string {
	u, err := url.Parse(address)
	if err != nil {
		return ""
	}

	if u.Host == "" {
		if strings.HasPrefix(u.Path, "/") {
			address = fmt.Sprintf("%s://%s%s", d.domain.Scheme, d.domain.Host, address)
		} else {
			address = fmt.Sprintf("%s://%s/%s", d.domain.Scheme, d.domain.Host, address)
		}
	}

	return address
}

func (d *defaultExtractor) isFileUrl(address string) bool {
	u, err := url.Parse(address)
	if err != nil {
		return false
	}

	return !strings.HasSuffix(u.Path, ".html") && !strings.HasSuffix(u.Path, ".htm") && d.fileRegex.MatchString(u.Path)
}
