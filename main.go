package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"strconv"
	"time"
)

func main() {
	const RFC2822 = "Mon Jan 02 15:04:05 -0700 2006"
	const BASE_URL = "https://unixtimestamps.herokuapp.com"
	const SITEMAP_JUMP = 100000

	r := gin.Default()
	r.LoadHTMLGlob("templates/*")

	r.GET("/", func(c *gin.Context) {
		c.String(200, "pong")
	})

	r.GET("/robots.txt", func(c *gin.Context) {
		c.String(200, "allow: *\nSitemap: %s/sitemap.txt", BASE_URL)
	})

	r.GET("/sitemap.txt", func(c *gin.Context) {
		out := ""
		i := 0
		for i < 2147472000 {
			i += SITEMAP_JUMP
			out += fmt.Sprintf("%s/sitemap/%d/%d/map.txt\n", BASE_URL, i, i+SITEMAP_JUMP)
		}

		c.String(200, out)
	})

	r.GET("/sitemap/:from/:to/map.txt", func(c *gin.Context) {
		from, _ := strconv.ParseInt(c.Param("from"), 10, 64)
		to, _ := strconv.ParseInt(c.Param("to"), 10, 64)

		out := ""

		//c.Stream(step)

		i := from
		for i < to {
			i += 1
			out += fmt.Sprintf("%s/unixtimestamp/%d\n", BASE_URL, i)
		}

		c.String(200, out)
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

	r.Run(":8080") // listen and serve on 0.0.0.0:8080
}
