package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// SeoData is a struct of useful SEO data
type SeoData struct {
	URL             string
	Title           string
	H1              string
	MetaDescription string
	StatusCode      int
}

// Parser defines the parsing interface
type Parser interface {
	GetSeoData(resp *http.Response) (SeoData, error)
}

// DefaultParser is en empty struct for implmenting default parser
type DefaultParser struct {
}

var userAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/61.0.3163.100 Safari/537.36",
	"Mozilla/5.0 (Windows NT 6.1; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/61.0.3163.100 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/61.0.3163.100 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_6) AppleWebKit/604.1.38 (KHTML, like Gecko) Version/11.0 Safari/604.1.38",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:56.0) Gecko/20100101 Firefox/56.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_13) AppleWebKit/604.1.38 (KHTML, like Gecko) Version/11.0 Safari/604.1.38",
}

var (
	baseURLFlag = flag.String("baseurl", "https://example.com", "Base URL of the site to find and scrape the sitemap")
)

func randomUserAgent() string {
	rand.Seed(time.Now().Unix())
	randNum := rand.Int() % len(userAgents)
	return userAgents[randNum]
}

func isSitemap(urls []string) ([]string, []string) {
	sitemapFiles := []string{}
	pages := []string{}
	for _, page := range urls {
		foundSitemap := strings.Contains(page, "xml")
		if foundSitemap {
			fmt.Println("Found Sitemap", page)
			sitemapFiles = append(sitemapFiles, page)
		} else {
			pages = append(pages, page)
		}
	}
	return sitemapFiles, pages
}

func extractSitemapURLs(startURL string) []string {
	worklist := make(chan []string)
	toCrawl := []string{}
	var n int
	n++
	go func() { worklist <- []string{startURL} }()
	for ; n > 0; n-- {
		list := <-worklist
		for _, link := range list {
			n++
			go func(link string) {
				response, err := makeRequest(link)
				if err != nil {
					log.Printf("Error retrieving URL: %s", link)
				}
				urls, _ := extractUrls(response)
				if err != nil {
					log.Printf("Error extracting document from response, URL: %s", link)
				}
				sitemapFiles, pages := isSitemap(urls)
				if sitemapFiles != nil {
					worklist <- sitemapFiles
				}
				toCrawl = append(toCrawl, pages...)
			}(link)
		}
	}
	return toCrawl
}

func makeRequest(url string) (*http.Response, error) {
	client := http.Client{
		Timeout: 10 * time.Second,
	}
	req, err := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", randomUserAgent())
	if err != nil {
		return nil, err
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func scrapeUrls(urls []string, parser Parser, concurrency int) []SeoData {
	tokens := make(chan struct{}, concurrency)
	var n int
	n++
	worklist := make(chan []string)
	results := []SeoData{}
	go func() { worklist <- urls }()
	for ; n > 0; n-- {
		list := <-worklist
		for _, url := range list {
			if url != "" {
				n++
				go func(url string, token chan struct{}) {
					log.Printf("Requesting URL: %s", url)
					res, err := scrapePage(url, tokens, parser)
					if err != nil {
						log.Printf("Encountered error, URL: %s", url)
					} else {
						results = append(results, res)
					}
					worklist <- []string{}
				}(url, tokens)
			}
		}
	}
	return results
}

func extractUrls(response *http.Response) ([]string, error) {
	doc, err := goquery.NewDocumentFromResponse(response)
	if err != nil {
		return nil, err
	}
	results := []string{}
	sel := doc.Find("loc")
	for i := range sel.Nodes {
		loc := sel.Eq(i)
		result := loc.Text()
		results = append(results, result)
	}
	return results, nil
}

func scrapePage(url string, token chan struct{}, parser Parser) (SeoData, error) {
	res, err := crawlPage(url, token)
	if err != nil {
		return SeoData{}, err
	}
	data, err := parser.GetSeoData(res)
	if err != nil {
		return SeoData{}, err
	}
	return data, nil
}

func crawlPage(url string, tokens chan struct{}) (*http.Response, error) {
	tokens <- struct{}{}
	resp, err := makeRequest(url)
	<-tokens
	if err != nil {
		return nil, err
	}
	return resp, err
}

// GetSeoData concrete implementation of the default parser
func (d DefaultParser) GetSeoData(resp *http.Response) (SeoData, error) {
	doc, err := goquery.NewDocumentFromResponse(resp)
	if err != nil {
		return SeoData{}, err
	}
	result := SeoData{}
	result.URL = resp.Request.URL.String()
	result.StatusCode = resp.StatusCode
	result.Title = doc.Find("title").First().Text()
	result.H1 = doc.Find("h1").First().Text()
	result.MetaDescription, _ = doc.Find("meta[name^=description]").Attr("content")
	return result, nil
}

// ScrapeSitemap scrapes a given sitemap
func ScrapeSitemap(url string, parser Parser, concurrency int) []SeoData {
	results := extractSitemapURLs(url)
	res := scrapeUrls(results, parser, concurrency)
	return res
}

func findSitemap(baseURL string) []string {
	commonPaths := []string{
		"/sitemap.xml",
		"/sitemap_index.xml",
		"/sitemap1.xml",
		"/robots.txt",
	}
	var sitemaps []string

	for _, path := range commonPaths {
		fullURL := baseURL + path
		resp, err := makeRequest(fullURL)
		if err != nil {
			log.Println("Error making request to:", fullURL, "Error:", err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			continue
		}

		if path == "/robots.txt" {
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Println("Error reading robots.txt body:", err)
				continue
			}
			sitemaps = append(sitemaps, parseRobotsTxt(string(body), baseURL)...)
		} else {
			sitemaps = append(sitemaps, fullURL)
		}
	}
	return sitemaps
}

// parseRobotsTxt parses the contents of robots.txt to find sitemap URLs
func parseRobotsTxt(contents, baseURL string) []string {
	var sitemaps []string
	lines := strings.Split(contents, "\n")
	for _, line := range lines {
		if strings.HasPrefix(strings.ToLower(line), "sitemap:") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				sitemapURL := strings.TrimSpace(parts[1])
				if !strings.HasPrefix(sitemapURL, "http") {
					sitemapURL = baseURL + sitemapURL
				}
				sitemaps = append(sitemaps, sitemapURL)
			}
		}
	}
	return sitemaps
}

func main() {
	flag.Parse()

	baseURL := *baseURLFlag
	if !strings.HasPrefix(baseURL, "http") {
		baseURL = "https://" + baseURL
	}

	// Ensure baseURL is properly parsed to append paths correctly
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		log.Fatalf("Invalid URL format: %s", err)
	}
	baseURL = parsedURL.Scheme + "://" + parsedURL.Host

	fmt.Println("Searching for sitemaps...")
	sitemapURLs := findSitemap(baseURL)

	if len(sitemapURLs) == 0 {
		fmt.Println("No sitemaps found.")
		return
	}

	p := DefaultParser{}
	results := []SeoData{}
	for _, sitemapURL := range sitemapURLs {
		fmt.Println("Scraping sitemap:", sitemapURL)
		results = append(results, ScrapeSitemap(sitemapURL, p, 10)...)
	}

	for _, res := range results {
		fmt.Println(res)
	}
}
