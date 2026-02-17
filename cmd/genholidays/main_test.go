package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	retryBaseDelay = 0 // Eliminate sleep in retry loops for all tests.
	os.Exit(m.Run())
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newHTTPResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func mustReadAll(t *testing.T, r io.Reader) string {
	t.Helper()
	b, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("reading response body failed: %v", err)
	}
	return string(b)
}

// --- validateCSVURL ---

func TestValidateCSVURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"allowed host syukujitsu", "https://www8.cao.go.jp/chosei/shukujitsu/syukujitsu.csv", false},
		{"allowed host shukujitsu", "https://www8.cao.go.jp/chosei/shukujitsu/shukujitsu.csv", false},
		{"allowed host www.cao.go.jp", "https://www.cao.go.jp/some/path.csv", false},
		{"blocked evil host", "https://evil.example.com/syukujitsu.csv", true},
		{"blocked localhost", "https://localhost/syukujitsu.csv", true},
		{"blocked internal IP", "https://192.168.1.1/syukujitsu.csv", true},
		{"blocked similar domain", "https://www8.cao.go.jp.evil.com/syukujitsu.csv", true},
		{"blocked HTTP", "http://www8.cao.go.jp/chosei/shukujitsu/syukujitsu.csv", true},
		{"blocked FTP", "ftp://www8.cao.go.jp/chosei/shukujitsu/syukujitsu.csv", true},
		{"blocked empty URL", "", true},
		{"blocked no scheme", "www8.cao.go.jp/chosei/shukujitsu/syukujitsu.csv", true},
		{"invalid URL parse", "://invalid", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCSVURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateCSVURL(%q) error = %v, wantErr = %v", tt.url, err, tt.wantErr)
			}
		})
	}
}

// --- parseCSV ---

func TestParseCSV_Valid(t *testing.T) {
	t.Parallel()

	csv := "国民の祝日・休日月日,国民の祝日・休日名称\r\n2024/1/1,元日\r\n2024/1/8,成人の日\r\n"
	holidays, err := parseCSV(strings.NewReader(csv))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(holidays) != 2 {
		t.Fatalf("expected 2 holidays, got %d", len(holidays))
	}
	if holidays[0].name != "元日" {
		t.Errorf("first holiday = %q, want 元日", holidays[0].name)
	}
	if holidays[0].year != 2024 || holidays[0].month != time.January || holidays[0].day != 1 {
		t.Errorf("first holiday date = %d/%d/%d, want 2024/1/1",
			holidays[0].year, holidays[0].month, holidays[0].day)
	}
}

func TestParseCSV_InvalidHeader(t *testing.T) {
	t.Parallel()

	csv := "date,name\r\n2024/1/1,元日\r\n"
	_, err := parseCSV(strings.NewReader(csv))
	if err == nil {
		t.Fatal("expected error for invalid header")
	}
	if !strings.Contains(err.Error(), "国民の祝日") {
		t.Errorf("error should mention expected header, got: %v", err)
	}
}

func TestParseCSV_InvalidDate(t *testing.T) {
	t.Parallel()

	csv := "国民の祝日月日,国民の祝日名称\r\nnot-a-date,元日\r\n"
	_, err := parseCSV(strings.NewReader(csv))
	if err == nil {
		t.Fatal("expected error for invalid date")
	}
	if !strings.Contains(err.Error(), "invalid date") {
		t.Errorf("error should mention invalid date, got: %v", err)
	}
}

func TestParseCSV_TooFewColumns(t *testing.T) {
	t.Parallel()

	csv := "国民の祝日月日,国民の祝日名称\r\n2024/1/1\r\n"
	_, err := parseCSV(strings.NewReader(csv))
	if err == nil {
		t.Fatal("expected error for too few columns")
	}
}

func TestParseCSV_EmptyRows(t *testing.T) {
	t.Parallel()

	csv := "国民の祝日月日,国民の祝日名称\r\n2024/1/1,元日\r\n,\r\n2024/5/3,憲法記念日\r\n"
	holidays, err := parseCSV(strings.NewReader(csv))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(holidays) != 2 {
		t.Errorf("expected 2 holidays (skipping empty row), got %d", len(holidays))
	}
}

func TestParseCSV_TooFewHeaderColumns(t *testing.T) {
	t.Parallel()

	csv := "国民の祝日月日\r\n2024/1/1,元日\r\n"
	_, err := parseCSV(strings.NewReader(csv))
	if err == nil {
		t.Fatal("expected error for single-column header")
	}
	if !strings.Contains(err.Error(), "unexpected header columns") {
		t.Errorf("error should mention column count, got: %v", err)
	}
}

func TestParseCSV_EmptyInput(t *testing.T) {
	t.Parallel()

	_, err := parseCSV(strings.NewReader(""))
	if err == nil {
		t.Fatal("expected error for empty input")
	}
	if !strings.Contains(err.Error(), "reading header") {
		t.Errorf("error should mention reading header, got: %v", err)
	}
}

func TestParseCSV_PartialEmptyFields(t *testing.T) {
	t.Parallel()

	csv := "国民の祝日月日,国民の祝日名称\r\n,元日\r\n2024/1/1,元日\r\n"
	holidays, err := parseCSV(strings.NewReader(csv))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(holidays) != 1 {
		t.Errorf("expected 1 holiday, got %d", len(holidays))
	}
}

// --- generate ---

func TestGenerate(t *testing.T) {
	t.Parallel()

	holidays := []holiday{
		{2024, time.May, 3, "憲法記念日"},
		{2024, time.January, 1, "元日"},
	}

	src, err := generate(holidays)
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	code := string(src)
	if !strings.Contains(code, "Code generated by cmd/genholidays; DO NOT EDIT.") {
		t.Error("missing generated comment")
	}
	janIdx := strings.Index(code, "元日")
	mayIdx := strings.Index(code, "憲法記念日")
	if janIdx < 0 || mayIdx < 0 {
		t.Fatal("missing holiday names in output")
	}
	if janIdx > mayIdx {
		t.Error("holidays should be sorted by date")
	}
	if !strings.Contains(code, "time.January") || !strings.Contains(code, "time.May") {
		t.Error("should use time.Month constants")
	}
}

func TestGenerate_MultipleYears(t *testing.T) {
	t.Parallel()

	holidays := []holiday{
		{2025, time.January, 1, "元日"},
		{2024, time.January, 1, "元日"},
	}

	src, err := generate(holidays)
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	code := string(src)
	if !strings.Contains(code, "// 2024") || !strings.Contains(code, "// 2025") {
		t.Error("should contain year comments for multiple years")
	}
}

func TestGenerate_SortByMonthThenDay(t *testing.T) {
	t.Parallel()

	holidays := []holiday{
		{2024, time.March, 20, "春分の日"},
		{2024, time.January, 1, "元日"},
		{2024, time.January, 8, "成人の日"},
	}

	src, err := generate(holidays)
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	code := string(src)
	janIdx := strings.Index(code, "元日")
	marIdx := strings.Index(code, "春分の日")
	if janIdx > marIdx {
		t.Error("January should come before March")
	}
}

func TestMonthConstName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		month time.Month
		want  string
	}{
		{time.January, "time.January"},
		{time.February, "time.February"},
		{time.December, "time.December"},
	}
	for _, tt := range tests {
		if got := monthConstName(tt.month); got != tt.want {
			t.Errorf("monthConstName(%v) = %q, want %q", tt.month, got, tt.want)
		}
	}
}

// --- HTTP mock helpers ---

func newCKANResponseJSON(csvURL string) string {
	resp := ckanResponse{Success: true}
	resp.Result.Resources = []struct {
		URL    string `json:"url"`
		Format string `json:"format"`
	}{{URL: csvURL, Format: "CSV"}}
	b, _ := json.Marshal(resp)
	return string(b)
}

// closedServer returns the URL of an already-closed httptest server (connection refused).
func closedServerURL() string {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	ts.Close()
	return ts.URL
}

// --- resolveCSVURLFrom ---

func TestResolveCSVURLFrom_Success(t *testing.T) {
	t.Parallel()

	expectedURL := "https://www8.cao.go.jp/chosei/shukujitsu/syukujitsu.csv"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("User-Agent"); got != userAgent {
			t.Errorf("User-Agent = %q, want %q", got, userAgent)
		}
		fmt.Fprint(w, newCKANResponseJSON(expectedURL))
	}))
	defer ts.Close()

	got, err := resolveCSVURLFrom(ts.Client(), ts.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != expectedURL {
		t.Errorf("got %q, want %q", got, expectedURL)
	}
}

func TestResolveCSVURL_Wrapper(t *testing.T) {
	t.Parallel()

	expectedURL := "https://www8.cao.go.jp/chosei/shukujitsu/syukujitsu.csv"
	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.URL.String() != ckanAPIURL {
				t.Fatalf("unexpected request URL: %s", req.URL.String())
			}
			if got := req.Header.Get("User-Agent"); got != userAgent {
				t.Fatalf("User-Agent = %q, want %q", got, userAgent)
			}
			return newHTTPResponse(http.StatusOK, newCKANResponseJSON(expectedURL)), nil
		}),
	}

	got, err := resolveCSVURL(client)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != expectedURL {
		t.Errorf("got %q, want %q", got, expectedURL)
	}
}

func TestResolveCSVURLFrom_NonOKStatus(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	_, err := resolveCSVURLFrom(ts.Client(), ts.URL)
	if err == nil {
		t.Fatal("expected error for non-200 status")
	}
}

func TestResolveCSVURLFrom_SuccessFalse(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(ckanResponse{Success: false})
	}))
	defer ts.Close()

	_, err := resolveCSVURLFrom(ts.Client(), ts.URL)
	if err == nil {
		t.Fatal("expected error for success=false")
	}
}

func TestResolveCSVURLFrom_NoCSVResource(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := ckanResponse{Success: true}
		resp.Result.Resources = []struct {
			URL    string `json:"url"`
			Format string `json:"format"`
		}{{URL: "https://example.com/data.json", Format: "JSON"}}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	_, err := resolveCSVURLFrom(ts.Client(), ts.URL)
	if err == nil {
		t.Fatal("expected error for no CSV resource")
	}
}

func TestResolveCSVURLFrom_InvalidJSON(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "not json")
	}))
	defer ts.Close()

	_, err := resolveCSVURLFrom(ts.Client(), ts.URL)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestResolveCSVURLFrom_SSRFBlocked(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, newCKANResponseJSON("https://evil.example.com/data.csv"))
	}))
	defer ts.Close()

	_, err := resolveCSVURLFrom(ts.Client(), ts.URL)
	if err == nil {
		t.Fatal("expected error for SSRF-blocked URL")
	}
}

func TestResolveCSVURLFrom_NetworkError(t *testing.T) {
	t.Parallel()

	_, err := resolveCSVURLFrom(&http.Client{Timeout: 1 * time.Second}, closedServerURL())
	if err == nil {
		t.Fatal("expected error for network failure")
	}
}

// --- fetchWithRetry ---

func TestFetchWithRetry_Success(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("User-Agent"); got != userAgent {
			t.Errorf("User-Agent = %q, want %q", got, userAgent)
		}
		w.Write([]byte("data"))
	}))
	defer ts.Close()

	reader, etag, lastModified, notModified, err := fetchWithRetry(ts.Client(), ts.URL, "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if notModified {
		t.Fatal("unexpected not-modified response")
	}
	if etag != "" || lastModified != "" {
		t.Fatalf("unexpected validators: etag=%q lastModified=%q", etag, lastModified)
	}
	if reader == nil {
		t.Fatal("expected non-nil reader")
	}
	if got := mustReadAll(t, reader); got != "data" {
		t.Errorf("response body = %q, want %q", got, "data")
	}
}

func TestFetchWithRetry_404_NoRetry(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	_, _, _, _, err := fetchWithRetry(ts.Client(), ts.URL, "", "")
	if err == nil {
		t.Fatal("expected error for 404")
	}
}

func TestFetchWithRetry_ServerError_RetriesThenSucceeds(t *testing.T) {
	t.Parallel()

	attempts := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Write([]byte("data"))
	}))
	defer ts.Close()

	reader, _, _, _, err := fetchWithRetry(ts.Client(), ts.URL, "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reader == nil {
		t.Fatal("expected non-nil reader")
	}
	if got := mustReadAll(t, reader); got != "data" {
		t.Errorf("response body = %q, want %q", got, "data")
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestFetchWithRetry_AllRetriesFail_503(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer ts.Close()

	_, _, _, _, err := fetchWithRetry(ts.Client(), ts.URL, "", "")
	if err == nil {
		t.Fatal("expected error after all retries fail")
	}
}

func TestFetchWithRetry_429_RetriesThenSucceeds(t *testing.T) {
	t.Parallel()

	attempts := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 2 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Write([]byte("data"))
	}))
	defer ts.Close()

	reader, _, _, _, err := fetchWithRetry(ts.Client(), ts.URL, "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reader == nil {
		t.Fatal("expected non-nil reader")
	}
	if got := mustReadAll(t, reader); got != "data" {
		t.Errorf("response body = %q, want %q", got, "data")
	}
}

func TestFetchWithRetry_ConditionalGET_304(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("If-None-Match"); got != `"etag-1"` {
			t.Fatalf("If-None-Match = %q, want %q", got, `"etag-1"`)
		}
		if got := r.Header.Get("If-Modified-Since"); got != "Wed, 01 Jan 2025 00:00:00 GMT" {
			t.Fatalf("If-Modified-Since = %q, unexpected", got)
		}
		w.WriteHeader(http.StatusNotModified)
	}))
	defer ts.Close()

	reader, _, _, notModified, err := fetchWithRetry(
		ts.Client(),
		ts.URL,
		`"etag-1"`,
		"Wed, 01 Jan 2025 00:00:00 GMT",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !notModified {
		t.Fatal("expected not-modified=true")
	}
	if reader != nil {
		t.Fatal("reader should be nil on 304")
	}
}

func TestFetchCSV_Wrapper(t *testing.T) {
	t.Parallel()

	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			switch req.URL.String() {
			case ckanAPIURL:
				return newHTTPResponse(http.StatusOK, newCKANResponseJSON(fallbackURL1)), nil
			case fallbackURL1:
				return newHTTPResponse(http.StatusOK, "csvdata"), nil
			default:
				return newHTTPResponse(http.StatusNotFound, ""), nil
			}
		}),
	}

	result, err := fetchCSV(client)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.NotModified {
		t.Fatal("unexpected not-modified result")
	}
	if result.Reader == nil {
		t.Fatal("expected non-nil reader")
	}
	if got := mustReadAll(t, result.Reader); got != "csvdata" {
		t.Errorf("response body = %q, want %q", got, "csvdata")
	}
}

func TestFetchWithRetry_NetworkError(t *testing.T) {
	t.Parallel()

	_, _, _, _, err := fetchWithRetry(&http.Client{Timeout: 1 * time.Second}, closedServerURL(), "", "")
	if err == nil {
		t.Fatal("expected error for network failure")
	}
}

// --- resolveCSVURLWithRetry ---

func TestResolveCSVURLWithRetry_RetryableStatusThenSuccess(t *testing.T) {
	t.Parallel()

	attempts := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		fmt.Fprint(w, newCKANResponseJSON(fallbackURL1))
	}))
	defer ts.Close()

	got, err := resolveCSVURLWithRetry(ts.Client(), ts.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != fallbackURL1 {
		t.Fatalf("got %q, want %q", got, fallbackURL1)
	}
	if attempts != 3 {
		t.Fatalf("attempts = %d, want 3", attempts)
	}
}

func TestResolveCSVURLWithRetry_NonRetryableStatus_NoRetry(t *testing.T) {
	t.Parallel()

	attempts := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	_, err := resolveCSVURLWithRetry(ts.Client(), ts.URL)
	if err == nil {
		t.Fatal("expected error")
	}
	if attempts != 1 {
		t.Fatalf("attempts = %d, want 1", attempts)
	}
}

// --- fetchCSVWithFallbacks ---

func TestFetchCSVWithFallbacks_CKANFails_Fb1Succeeds(t *testing.T) {
	t.Parallel()

	fb1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("csvdata"))
	}))
	defer fb1.Close()

	ckan := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ckan.Close()

	fb2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer fb2.Close()

	result, err := fetchCSVWithFallbacks(&http.Client{Timeout: 5 * time.Second}, ckan.URL, fb1.URL, fb2.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Reader == nil {
		t.Fatal("expected non-nil reader")
	}
	if got := mustReadAll(t, result.Reader); got != "csvdata" {
		t.Errorf("response body = %q, want %q", got, "csvdata")
	}
}

func TestFetchCSVWithFallbacks_CKANResolvesToSameAsFb1(t *testing.T) {
	t.Parallel()

	// When CKAN resolves to the same URL as fb1, deduplication prevents trying it twice.
	// We use a mock fb1 server that returns 404, and CKAN resolves to fb1's URL.
	fb1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer fb1.Close()

	ckan := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Resolve to fb1's URL (same as the first fallback).
		// validateCSVURL will reject non-allowed hosts, so CKAN resolution fails.
		// This tests the fallback path when CKAN returns an invalid URL.
		fmt.Fprint(w, newCKANResponseJSON(fb1.URL)) // localhost not in allowedCSVHosts
	}))
	defer ckan.Close()

	fb2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("csvdata"))
	}))
	defer fb2.Close()

	result, err := fetchCSVWithFallbacks(&http.Client{Timeout: 5 * time.Second}, ckan.URL, fb1.URL, fb2.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Reader == nil {
		t.Fatal("expected non-nil reader")
	}
	if got := mustReadAll(t, result.Reader); got != "csvdata" {
		t.Errorf("response body = %q, want %q", got, "csvdata")
	}
}

func TestFetchCSVWithFallbacks_AllFail(t *testing.T) {
	t.Parallel()

	failServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer failServer.Close()

	ckan := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ckan.Close()

	_, err := fetchCSVWithFallbacks(&http.Client{Timeout: 5 * time.Second}, ckan.URL, failServer.URL, failServer.URL)
	if err == nil {
		t.Fatal("expected error when all URLs fail")
	}
	if !strings.Contains(err.Error(), "all URLs failed") {
		t.Errorf("error should mention all URLs failed, got: %v", err)
	}
}

func TestFetchCSVWithFallbacks_Fb1Fails_Fb2Succeeds(t *testing.T) {
	t.Parallel()

	fail := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer fail.Close()

	ok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("csvdata"))
	}))
	defer ok.Close()

	ckan := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ckan.Close()

	result, err := fetchCSVWithFallbacks(&http.Client{Timeout: 5 * time.Second}, ckan.URL, fail.URL, ok.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Reader == nil {
		t.Fatal("expected non-nil reader")
	}
	if got := mustReadAll(t, result.Reader); got != "csvdata" {
		t.Errorf("response body = %q, want %q", got, "csvdata")
	}
}
