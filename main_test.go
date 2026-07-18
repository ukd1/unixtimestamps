package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func TestSitemapRange(t *testing.T) {
	r := testRouter()
	w := performRequest(r, "/sitemaps/0/3/map.txt", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %q", w.Code, http.StatusOK, w.Body.String())
	}

	want := baseURL + "/unixtimestamp/0\n" +
		baseURL + "/unixtimestamp/1\n" +
		baseURL + "/unixtimestamp/2\n"
	if got := w.Body.String(); got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestSitemapRejectsInvalidRanges(t *testing.T) {
	r := testRouter()
	paths := []string{
		"/sitemaps/not-a-number/3/map.txt",
		"/sitemaps/-1/3/map.txt",
		"/sitemaps/3/3/map.txt",
		"/sitemaps/4/3/map.txt",
		"/sitemaps/0/50001/map.txt",
		"/sitemaps/2147483647/2147483649/map.txt",
	}

	for _, path := range paths {
		w := performRequest(r, path, nil)
		if w.Code != http.StatusBadRequest {
			t.Errorf("GET %s status = %d, want %d", path, w.Code, http.StatusBadRequest)
		}
	}
}

func TestSitemapIndexCoversCompleteSigned32BitRange(t *testing.T) {
	r := testRouter()
	w := performRequest(r, "/sitemap.xml", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	body := w.Body.String()
	first := "<sitemap><loc>" + baseURL + "/sitemaps/0/50000/map.txt</loc>"
	if index := strings.Index(body, "<sitemap><loc>"); index < 0 || !strings.HasPrefix(body[index:], first) {
		t.Fatalf("first sitemap entry does not cover timestamps starting at zero")
	}

	last := "<sitemap><loc>" + baseURL + "/sitemaps/2147450000/2147483648/map.txt</loc>"
	if !strings.Contains(body, last) {
		t.Fatalf("sitemap index does not contain final range %q", last)
	}
}

func TestTimestampValidation(t *testing.T) {
	r := testRouter()
	paths := []string{
		"/unixtimestamp/not-a-number",
		"/unixtimestamp/-1",
		"/unixtimestamp/253402300800",
		"/unixtimestamp/9223372036854775807",
	}

	for _, path := range paths {
		w := performRequest(r, path, nil)
		if w.Code != http.StatusBadRequest {
			t.Errorf("GET %s status = %d, want %d", path, w.Code, http.StatusBadRequest)
		}
	}
}

func TestTimestampUsesUTCAndRFC2822(t *testing.T) {
	r := testRouter()
	w := performRequest(r, "/unixtimestamp/0", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %q", w.Code, http.StatusOK, w.Body.String())
	}

	body := w.Body.String()
	for _, want := range []string{
		"1970-01-01T00:00:00Z",
		"Thu, 01 Jan 1970 00:00:00 +0000",
		"Beginning",
		"/unixtimestamp/1",
		"/unixtimestamp/50000",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body does not contain %q", want)
		}
	}
	if strings.Contains(body, "/unixtimestamp/-1") {
		t.Error("minimum timestamp page contains an overflowing previous link")
	}
}

func TestTimestampNavigationAtJumpBoundary(t *testing.T) {
	r := testRouter()
	w := performRequest(r, "/unixtimestamp/49999", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if strings.Contains(w.Body.String(), "/unixtimestamp/100000") {
		t.Error("timestamp immediately before a jump skips the 50000 boundary")
	}
}

func TestMaximumTimestampHasNoOverflowingLinks(t *testing.T) {
	r := testRouter()
	w := performRequest(r, "/unixtimestamp/253402300799", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	body := w.Body.String()
	if !strings.Contains(body, "End") {
		t.Error("maximum timestamp page does not mark the end of the supported range")
	}
	if strings.Contains(body, "253402300800") {
		t.Error("maximum timestamp page contains an out-of-range next link")
	}
}

func TestRequestLogFieldsUseRequestHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Request.Header.Set("CF-RAY", "ray-id")
	c.Request.Header.Set("CF-IPCountry", "US")
	c.Request.Header.Set("CF-Connecting-IP", "203.0.113.8")

	fields := requestLogFields(c)
	got := make(map[string]string, len(fields))
	for _, field := range fields {
		got[field.Key] = field.String
	}

	want := map[string]string{
		"cf_ray":        "ray-id",
		"cf_ip_country": "US",
		"real_ip":       "203.0.113.8",
	}
	for key, value := range want {
		if got[key] != value {
			t.Errorf("field %q = %q, want %q", key, got[key], value)
		}
	}
}

func testRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return newRouter(zap.NewNop())
}

func performRequest(r http.Handler, path string, headers map[string]string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	r.ServeHTTP(w, req)
	return w
}
