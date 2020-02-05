package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"os"
	"strconv"
	"time"
)

func main() {
	const RFC2822 = "Mon Jan 02 15:04:05 -0700 2006"
	const BASE_URL = "https://unixtimestamps.rsmith.co"
	const SITEMAP_JUMP = 50000

	r := gin.Default()
	r.LoadHTMLGlob("templates/*")

	r.GET("/", func(c *gin.Context) {
		c.HTML(200, "index.html", gin.H{})
	})

	r.GET("/robots.txt", func(c *gin.Context) {
		c.String(200, "allow: *\nSitemap: %s/sitemap.xml", BASE_URL)
	})

	r.GET("/sitemap.xml", func(c *gin.Context) {
		c.Stream(func(w io.Writer) bool {
			year, month, day := time.Now().Date()
			t := fmt.Sprintf("%04d-%02d-%02d", year, month, day)

			w.Write([]byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?>"))
			w.Write([]byte("<sitemapindex xmlns=\"http://www.sitemaps.org/schemas/sitemap/0.9\">"))

			// start from somewhere in the middle, as we can only have 500 files
			i := 0
			for i < 2147472000 {
				i += SITEMAP_JUMP
				w.Write([]byte(fmt.Sprintf("\n<sitemap>\n <loc>%s/sitemaps/%d/%d/map.txt</loc>\n <lastmod>%s</lastmod>\n</sitemap>", BASE_URL, i, i+SITEMAP_JUMP, t)))
			}

			w.Write([]byte("</sitemapindex>"))
			return false
		})
	})

	r.GET("/sitemaps/:from/:to/map.txt", func(c *gin.Context) {
		from, _ := strconv.ParseInt(c.Param("from"), 10, 64)
		to, _ := strconv.ParseInt(c.Param("to"), 10, 64)

		c.Stream(func(w io.Writer) bool {
			i := from
			for i < to {
				i += 1
				w.Write([]byte(fmt.Sprintf("%s/unixtimestamp/%d\n", BASE_URL, i)))
			}

			return false
		})
	})

	r.GET("/unixtimestamp/:uts", func(c *gin.Context) {
		i, err := strconv.ParseInt(c.Param("uts"), 10, 64)
		if err != nil {
			panic(err)
		}

		t := time.Unix(i, 0)

		c.HTML(200, "time.html", gin.H{
			"ts_unix":     i,
			"ts_unix_pp":  i + 1,
			"ts_unix_mm":  i - 1,
			"ts_unix_jpp": (((i + 1) / SITEMAP_JUMP) * SITEMAP_JUMP) + SITEMAP_JUMP,
			"ts_unix_jmm": ((i - 1) / SITEMAP_JUMP) * SITEMAP_JUMP,
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
			"RFC2822":     t.Format(RFC2822),
		})
	})

	if os.Getenv("PORT") != "" {
		r.Run(":" + os.Getenv("PORT"))
	} else {
		r.Run(":8080")
	}
}
