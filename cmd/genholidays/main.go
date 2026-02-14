// Command genholidays fetches the Japanese national holiday CSV from the
// Cabinet Office website and generates a Go source file containing the
// holiday data as a map literal.
//
// The CSV URL is resolved dynamically via the e-Gov Data Portal CKAN API
// (recommended by the Digital Agency of Japan). If the API is unavailable,
// it falls back to well-known direct URLs.
//
// Usage:
//
//	go run main.go -output ../../holidays_data.go
package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"go/format"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
)

const (
	// CKAN API endpoint for the holiday dataset (recommended by Digital Agency).
	ckanAPIURL = "https://data.e-gov.go.jp/data/api/action/package_show?id=cao_20190522_0002"

	// Fallback CSV URLs in case the CKAN API is unavailable.
	fallbackURL1 = "https://www8.cao.go.jp/chosei/shukujitsu/syukujitsu.csv"
	fallbackURL2 = "https://www8.cao.go.jp/chosei/shukujitsu/shukujitsu.csv"

	minExpectedRows = 1000

	httpTimeout = 30 * time.Second
	maxRetries  = 3

	// Maximum response sizes to prevent memory exhaustion.
	maxJSONResponseSize = 1 * 1024 * 1024 // 1 MB for CKAN API response
	maxCSVResponseSize  = 5 * 1024 * 1024 // 5 MB for CSV data

	userAgent = "jp-holidays-generator/1.0 (https://github.com/rabitt1ove/jp-holidays)"
)

// retryBaseDelay is the base delay between retry attempts (variable for testing).
var retryBaseDelay = 2 * time.Second

// allowedCSVHosts is the set of hostnames allowed for CSV download URLs.
// This prevents SSRF if the CKAN API returns an unexpected URL.
var allowedCSVHosts = map[string]bool{
	"www8.cao.go.jp": true,
	"www.cao.go.jp":  true,
}

// ckanResponse represents the relevant parts of the CKAN API response.
type ckanResponse struct {
	Success bool `json:"success"`
	Result  struct {
		Resources []struct {
			URL    string `json:"url"`
			Format string `json:"format"`
		} `json:"resources"`
	} `json:"result"`
}

type holiday struct {
	year  int
	month time.Month
	day   int
	name  string
}

func main() {
	output := flag.String("output", "holidays_data.go", "output file path")
	flag.Parse()

	log.SetFlags(0)
	log.SetPrefix("genholidays: ")

	client := &http.Client{Timeout: httpTimeout}

	body, err := fetchCSV(client)
	if err != nil {
		log.Fatalf("failed to fetch CSV: %v", err)
	}

	holidays, err := parseCSV(body)
	if err != nil {
		log.Fatalf("failed to parse CSV: %v", err)
	}

	if len(holidays) < minExpectedRows {
		log.Fatalf("validation failed: expected at least %d rows, got %d", minExpectedRows, len(holidays))
	}

	src, err := generate(holidays)
	if err != nil {
		log.Fatalf("failed to generate source: %v", err)
	}

	if err := os.WriteFile(*output, src, 0644); err != nil {
		log.Fatalf("failed to write output: %v", err)
	}

	log.Printf("wrote %d holidays to %s", len(holidays), *output)
}

// resolveCSVURL queries the CKAN API to get the current CSV download URL.
func resolveCSVURL(client *http.Client) (string, error) {
	return resolveCSVURLFrom(client, ckanAPIURL)
}

// resolveCSVURLFrom queries the given CKAN API endpoint to get the current CSV download URL.
func resolveCSVURLFrom(client *http.Client, apiURL string) (string, error) {
	log.Printf("resolving CSV URL via CKAN API: %s", apiURL)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, apiURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("CKAN API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("CKAN API returned status %d", resp.StatusCode)
	}

	var ckan ckanResponse
	limited := io.LimitReader(resp.Body, maxJSONResponseSize)
	if err := json.NewDecoder(limited).Decode(&ckan); err != nil {
		return "", fmt.Errorf("CKAN API response decode failed: %w", err)
	}

	if !ckan.Success {
		return "", fmt.Errorf("CKAN API returned success=false")
	}

	for _, r := range ckan.Result.Resources {
		if strings.EqualFold(r.Format, "CSV") && r.URL != "" {
			// Validate the URL host to prevent SSRF.
			if err := validateCSVURL(r.URL); err != nil {
				return "", fmt.Errorf("CKAN returned invalid URL: %w", err)
			}
			log.Printf("  resolved URL: %s", r.URL)
			return r.URL, nil
		}
	}

	return "", fmt.Errorf("no CSV resource found in CKAN response")
}

// validateCSVURL checks that a URL points to an allowed host (SSRF prevention).
func validateCSVURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL %q: %w", rawURL, err)
	}
	if parsed.Scheme != "https" {
		return fmt.Errorf("URL %q: only HTTPS is allowed", rawURL)
	}
	if !allowedCSVHosts[parsed.Hostname()] {
		return fmt.Errorf("URL %q: host %q is not in the allowed list", rawURL, parsed.Hostname())
	}
	return nil
}

// fetchCSV resolves the CSV URL and fetches it with retries.
// Strategy: CKAN API -> fallback URL 1 -> fallback URL 2.
func fetchCSV(client *http.Client) (io.Reader, error) {
	return fetchCSVWithFallbacks(client, ckanAPIURL, fallbackURL1, fallbackURL2)
}

// fetchCSVWithFallbacks resolves the CSV URL via the given CKAN API and fetches it with retries.
func fetchCSVWithFallbacks(client *http.Client, ckanURL, fb1, fb2 string) (io.Reader, error) {
	// Build ordered list of URLs to try.
	var urls []string

	// Try CKAN API first.
	if resolved, err := resolveCSVURLFrom(client, ckanURL); err != nil {
		log.Printf("  CKAN API failed: %v (falling back to direct URLs)", err)
	} else {
		urls = append(urls, resolved)
	}

	// Add fallback URLs (skip if CKAN already resolved to same URL).
	for _, fb := range []string{fb1, fb2} {
		if len(urls) == 0 || urls[0] != fb {
			urls = append(urls, fb)
		}
	}

	var lastErr error
	for _, url := range urls {
		reader, err := fetchWithRetry(client, url)
		if err != nil {
			lastErr = err
			continue
		}
		return reader, nil
	}
	return nil, fmt.Errorf("all URLs failed, last error: %w", lastErr)
}

// fetchWithRetry fetches a URL with exponential backoff retries.
func fetchWithRetry(client *http.Client, url string) (io.Reader, error) {
	var lastErr error
	for attempt := range maxRetries {
		if attempt > 0 {
			delay := retryBaseDelay * time.Duration(1<<(attempt-1))
			log.Printf("  retrying in %v (attempt %d/%d)", delay, attempt+1, maxRetries)
			time.Sleep(delay)
		}

		log.Printf("fetching %s", url)
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("creating request: %w", err)
		}
		req.Header.Set("User-Agent", userAgent)

		resp, err := client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("GET %s: %w", url, err)
			log.Printf("  failed: %v", err)
			continue
		}

		if resp.StatusCode == http.StatusTooManyRequests ||
			resp.StatusCode == http.StatusServiceUnavailable ||
			resp.StatusCode >= 500 {
			resp.Body.Close()
			lastErr = fmt.Errorf("GET %s: status %d", url, resp.StatusCode)
			log.Printf("  failed: status %d (retryable)", resp.StatusCode)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, fmt.Errorf("GET %s: status %d", url, resp.StatusCode)
		}

		limited := io.LimitReader(resp.Body, maxCSVResponseSize)
		decoder := japanese.ShiftJIS.NewDecoder()
		return transform.NewReader(limited, decoder), nil
	}
	return nil, lastErr
}

// parseCSV parses the Cabinet Office holiday CSV and validates its format.
func parseCSV(r io.Reader) ([]holiday, error) {
	reader := csv.NewReader(r)
	reader.LazyQuotes = true

	// Read and validate header.
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("reading header: %w", err)
	}
	if len(header) < 2 {
		return nil, fmt.Errorf("unexpected header columns: %d (expected 2)", len(header))
	}
	if !strings.Contains(header[0], "国民の祝日") {
		return nil, fmt.Errorf("unexpected header: %q (expected to contain '国民の祝日')", header[0])
	}

	var holidays []holiday
	lineNum := 1
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNum+1, err)
		}
		lineNum++

		if len(record) < 2 {
			return nil, fmt.Errorf("line %d: expected 2 columns, got %d", lineNum, len(record))
		}

		dateStr := strings.TrimSpace(record[0])
		name := strings.TrimSpace(record[1])

		if dateStr == "" || name == "" {
			continue
		}

		t, err := time.Parse("2006/1/2", dateStr)
		if err != nil {
			return nil, fmt.Errorf("line %d: invalid date %q: %w", lineNum, dateStr, err)
		}

		holidays = append(holidays, holiday{
			year:  t.Year(),
			month: t.Month(),
			day:   t.Day(),
			name:  name,
		})
	}

	return holidays, nil
}

// monthConstName returns the time.Month constant name (e.g., "time.January").
func monthConstName(m time.Month) string {
	return "time." + m.String()
}

// generate produces a formatted Go source file containing the holiday data.
func generate(holidays []holiday) ([]byte, error) {
	sort.Slice(holidays, func(i, j int) bool {
		if holidays[i].year != holidays[j].year {
			return holidays[i].year < holidays[j].year
		}
		if holidays[i].month != holidays[j].month {
			return holidays[i].month < holidays[j].month
		}
		return holidays[i].day < holidays[j].day
	})

	var b strings.Builder
	b.WriteString("// Code generated by cmd/genholidays; DO NOT EDIT.\n\n")
	b.WriteString("package jpholiday\n\n")
	b.WriteString("import \"time\"\n\n")
	b.WriteString("var builtinHolidays = map[date]string{\n")

	currentYear := 0
	for _, h := range holidays {
		if h.year != currentYear {
			if currentYear != 0 {
				b.WriteString("\n")
			}
			fmt.Fprintf(&b, "\t// %d\n", h.year)
			currentYear = h.year
		}
		fmt.Fprintf(&b, "\t{%d, %s, %d}: %q,\n", h.year, monthConstName(h.month), h.day, h.name)
	}

	b.WriteString("}\n")

	return format.Source([]byte(b.String()))
}
