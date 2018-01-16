package main

// Page struct represents a single crawled website.
// It holds references to pages that it links to and that link to it,
// as well as the list of static assets it depends on
type Page struct {
	Title, Url          string
	LinksTo, LinkedFrom []*Page
	Assets              []*Asset
}

type Asset struct {
	Type AssetType
	Url  string
}

type AssetType uint8

const (
	Link   AssetType = iota
	Script AssetType = iota
	Image  AssetType = iota
	Video  AssetType = iota
)

type result struct {
	url, from string
	body      []byte
}
