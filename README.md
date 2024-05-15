# Sitemap Crawler for SEO (Search Engine Optimization)

A Go-based sitemap crawler that automates the extraction of SEO data from websites. It is designed to efficiently retrieve URLs, page titles, H1 tags, and meta descriptions from the sitemaps of specified websites.

## Features

- **Automatic Sitemap Discovery**: Discovers and parses sitemaps from a given base URL.
- **SEO Data Extraction**: Retrieves crucial SEO metrics such as page titles, H1 tags, and meta descriptions.
- **Concurrency Support**: Manages multiple URLs concurrently to speed up the crawling process.
- **robots.txt**: Adheres to the directives specified in `robots.txt` files to ensure compliant web scraping.

## Getting Started

### Prerequisites

- Go (Golang) installed on your machine.

### Installation

Clone the repository to your local machine:

```bash
git clone https://github.com/Dev-29/sitemap-crawler.git
cd sitemap-crawler
```
### Usage

Run the program with the following command:

```bash
go run main.go -baseurl "https://example.com/"
```
