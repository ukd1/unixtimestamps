package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	rfc2822                   = "Mon, 02 Jan 2006 15:04:05 -0700"
	baseURL                   = "https://unixtimestamps.rsmith.co"
	sitemapJump         int64 = 50_000
	sitemapMaxExclusive int64 = 1 << 31
	minUnixTimestamp    int64 = 0
	maxUnixTimestamp    int64 = 253_402_300_799 // 9999-12-31T23:59:59Z
)

var gitSHA = "unknown"

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		panic(fmt.Sprintf("create logger: %v", err))
	}
	defer func() { _ = logger.Sync() }()

	r := newRouter(logger)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if err := r.Run(":" + port); err != nil {
		logger.Fatal("run HTTP server", zap.Error(err))
	}
}

func newRouter(logger *zap.Logger) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(func(c *gin.Context) {
		c.Header("Content-Security-Policy", "default-src 'self';script-src 'self' 'unsafe-inline';style-src 'self' 'unsafe-inline';img-src data:;")
		c.Next()
	})
	r.Use(ginzap.GinzapWithConfig(logger, &ginzap.Config{
		UTC:        true,
		TimeFormat: time.RFC3339,
		Context:    ginzap.Fn(requestLogFields),
	}))

	r.LoadHTMLGlob("templates/*")

	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{"gitSHA": gitSHA})
	})

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	r.GET("/readiness", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	r.GET("/robots.txt", func(c *gin.Context) {
		c.String(http.StatusOK, "user-agent: *\nallow: /\nSitemap: %s/sitemap.xml", baseURL)
	})

	r.GET("/sitemap.xml", sitemapIndexHandler)
	r.GET("/sitemaps/:from/:to/map.txt", sitemapHandler)
	r.GET("/unixtimestamp/:uts", timestampHandler)

	return r
}

func requestLogFields(c *gin.Context) []zapcore.Field {
	fields := make([]zapcore.Field, 0, 3)

	cfRay := c.GetHeader("CF-RAY")
	if cfRay == "" {
		cfRay = "?"
	}
	fields = append(fields, zap.String("cf_ray", cfRay))

	cfIPCountry := c.GetHeader("CF-IPCountry")
	if cfIPCountry == "" {
		cfIPCountry = "?"
	}
	fields = append(fields, zap.String("cf_ip_country", cfIPCountry))

	clientIP := c.GetHeader("CF-Connecting-IPv6")
	if clientIP == "" {
		clientIP = c.GetHeader("CF-Connecting-IP")
	}
	if clientIP == "" {
		clientIP = c.GetHeader("X-Forwarded-For")
	}
	if clientIP == "" {
		clientIP = "?"
	}
	fields = append(fields, zap.String("real_ip", clientIP))

	return fields
}

func sitemapIndexHandler(c *gin.Context) {
	c.Header("Content-Type", "application/xml; charset=utf-8")
	c.Stream(func(w io.Writer) bool {
		if _, err := io.WriteString(w, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>"); err != nil {
			return false
		}
		if _, err := io.WriteString(w, "<sitemapindex xmlns=\"http://www.sitemaps.org/schemas/sitemap/0.9\">"); err != nil {
			return false
		}

		lastModified := time.Now().UTC().Format("2006-01-02")
		for from := int64(0); from < sitemapMaxExclusive; from += sitemapJump {
			if c.Request.Context().Err() != nil {
				return false
			}

			to := from + sitemapJump
			if to > sitemapMaxExclusive {
				to = sitemapMaxExclusive
			}
			if _, err := fmt.Fprintf(w, "<sitemap><loc>%s/sitemaps/%d/%d/map.txt</loc><lastmod>%s</lastmod></sitemap>", baseURL, from, to, lastModified); err != nil {
				return false
			}
		}

		_, _ = io.WriteString(w, "</sitemapindex>")
		return false
	})
}

func sitemapHandler(c *gin.Context) {
	from, fromErr := strconv.ParseInt(c.Param("from"), 10, 64)
	to, toErr := strconv.ParseInt(c.Param("to"), 10, 64)
	if fromErr != nil || toErr != nil || from < 0 || to <= from || to > sitemapMaxExclusive || to-from > sitemapJump {
		badRequest(c, "invalid sitemap range")
		return
	}

	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.Stream(func(w io.Writer) bool {
		for i := from; i < to; i++ {
			if c.Request.Context().Err() != nil {
				return false
			}
			if _, err := fmt.Fprintf(w, "%s/unixtimestamp/%d\n", baseURL, i); err != nil {
				return false
			}
		}
		return false
	})
}

func timestampHandler(c *gin.Context) {
	i, err := strconv.ParseInt(c.Param("uts"), 10, 64)
	if err != nil || i < minUnixTimestamp || i > maxUnixTimestamp {
		badRequest(c, "timestamp must be between 0 and 253402300799")
		return
	}

	t := time.Unix(i, 0).UTC()
	data := gin.H{
		"ts_unix":     i,
		"ANSIC":       t.Format(time.ANSIC),
		"UnixDate":    t.Format(time.UnixDate),
		"RubyDate":    t.Format(time.RubyDate),
		"RFC822":      t.Format(time.RFC822),
		"RFC822Z":     t.Format(time.RFC822Z),
		"RFC850":      t.Format(time.RFC850),
		"RFC1123":     t.Format(time.RFC1123),
		"RFC1123Z":    t.Format(time.RFC1123Z),
		"RFC3339":     t.Format(time.RFC3339),
		"RFC3339Nano": t.Format(time.RFC3339Nano),
		"RFC2822":     t.Format(rfc2822),
		"gitSHA":      gitSHA,
	}

	if i > minUnixTimestamp {
		previous := i - 1
		previousJump := (previous / sitemapJump) * sitemapJump
		data["has_prev"] = true
		data["ts_unix_mm"] = previous
		if previousJump < previous {
			data["has_prev_jump"] = true
			data["ts_unix_jmm"] = previousJump
		}
	}

	if i < maxUnixTimestamp {
		next := i + 1
		nextJump := ((i / sitemapJump) + 1) * sitemapJump
		data["has_next"] = true
		data["ts_unix_pp"] = next
		if nextJump > next && nextJump <= maxUnixTimestamp {
			data["has_next_jump"] = true
			data["ts_unix_jpp"] = nextJump
		}
	}

	c.HTML(http.StatusOK, "time.html", data)
}

func badRequest(c *gin.Context, message string) {
	c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": message})
}
