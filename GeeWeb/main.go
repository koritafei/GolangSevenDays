/**
 * @program: GolangGee
 * @description:实现简单web服务器
 * @author: koritafei
 * @create: 2021-04-28 15:08
 * @version v0.1
 * */

package main

import (
	"gee/gee"
	"log"
	"net/http"
	"time"
)

func ProcessRequest() gee.HandlerFunc {
	return func(c *gee.Context) {
		t := time.Now()
		log.Printf("middleware process request, +%v\n", c)
		c.Next()
		log.Printf("middleware process request, +%v,time %v\n", c, time.Since(t))
	}
}

func ProcessRequest2() gee.HandlerFunc {
	return func(c *gee.Context) {
		t := time.Now()
		log.Printf("middleware2 process request, +%v\n", c)
		c.Next()
		log.Printf("middleware2 process request, +%v,time %v\n", c, time.Since(t))
	}
}

func main() {
	r := gee.New()
	r.Use(gee.Logger())
	r.GET("/index", func(c *gee.Context) {
		c.HTML(http.StatusOK, "<br>Hello Gee!</br>", nil)
	})

	v1 := r.Group("/v1")
	v1.Use(ProcessRequest())
	{
		v1.GET("/", func(c *gee.Context) {
			c.HTML(http.StatusOK, "<h1>Hello Gee</h1>", nil)
		})

		v1.GET("/hello", func(c *gee.Context) {
			// expect /hello?name=geektutu
			c.String(http.StatusOK, "hello %s, you're at %s\n", c.Query("name"), c.Path)
		})
	}

	v2 := r.Group("/v2")
	v2.Use(ProcessRequest2())
	{
		v2.GET("/hello/:name", func(c *gee.Context) {
			// expect /hello/geektutu
			c.String(http.StatusOK, "hello %s, you're at %s\n", c.Param("name"), c.Path)
		})
		v2.POST("/login", func(c *gee.Context) {
			c.JSON(http.StatusOK, gee.H{
				"username": c.PostForm("username"),
				"password": c.PostForm("password"),
			})
		})

	}

	r.GET("/hello", func(c *gee.Context) {
		c.String(http.StatusOK, "hello %s, you are at %s", c.Query("name"), c.Path)
	})

	r.GET("/hello/:name", func(c *gee.Context) {
		c.String(http.StatusOK, "hello %s, you're at %s\n", c.Param("name"), c.Path)
	})

	r.GET("/assets/*filepath", func(c *gee.Context) {
		c.JSON(http.StatusOK, gee.H{"filepath": c.Param("filepath")})
	})

	r.Run(":8080")
}
