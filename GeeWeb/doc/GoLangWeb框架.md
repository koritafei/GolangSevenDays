# `WEB`框架
## 设计一个框架
在实现框架之前，我们需要回答框架解决的核心问题是什么。
在标准库的`net/http`中如何处理一个请求.
```go
// net/http 处理请求
package main

import (
	"fmt"
	"log"
	"net/http"
)

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "URL.Path = %q\n", r.URL.Path)
}

func counter(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "counter = %d\n", 1)
}

func main() {
	http.HandleFunc("/", handler)
	http.HandleFunc("/count", counter)
	log.Fatal(http.ListenAndServe("localhost:8080", nil))
}
```
`net/http`提供了基础的`Web`功能，即监听端口，映射静态路由，解析`http`报文等。一些`Web`开发中的简单需求并不支持，需要手动实现：
* 动态路由：例如`hello/:name`, `hello/*`之类的需求；
* 鉴权：没有分组/统一的鉴权能力，需要在每个路由映射的`Handler`中实现；
* 模板：没有统一简化的`HTML`机制；
* ...

考虑通用框架功能，主要包括以下核心能力：
* 路由(`Routing`): 将请求映射到函数，支持动态路由,如`hello/:name`；
* 模板(`Templates`): 使用内置模板机制提供模板渲染机制；
* 工具集(`Utilites`): 提供对`cookies`,`headers`等处理机制；
* 插件(`plugin`): 通过插件动态扩展功能。

## `HTTP`基础
`net/http`库的使用：
```go
/**
 * @program: GolangGee
 * @description:
 * @author: koritafei
 * @create: 2021-04-28 14:42
 * @version v0.1
 * */

package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/hello", helloHandler)
	log.Fatal(http.ListenAndServe(":9999", nil))
}


func indexHandler(w http.ResponseWriter, req *http.Request){
	fmt.Fprintf(w, "URL.path = %q\n", req.URL.Path)
}

func helloHandler(w http.ResponseWriter, req *http.Request) {
	for k,v := range req.Header {
		fmt.Fprintf(w, "Header[%q] = %q\n", k, v)
	}
}
```
### 实现`http.Handler`接口
```go
package http

import "net/http"

type Handler interface {
	ServeHTTP(w http.ResponseWriter, r *http.Request)
}

func ListenAndServe(address string, h Handler) error
```
利用`ServeHTTP`实现一个简单`Web`:
```go
package main

import (
	"fmt"
	"log"
	"net/http"
)

type Engine struct{}

func (engine *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	switch req.URL.Path {
	case "/":
		fmt.Fprintf(w, "URL Path = %s\n", req.URL.Path)
	case "/hello":
		for k, v := range req.Header {
			fmt.Fprintf(w, "Header[%q] = %q\n", k, v)
		}
	default:
		fmt.Fprintf(w, "404 NOT FOUND : %s\n", req.URL.Path)
	}
}

func main() {
	engine := new(Engine)
	log.Fatal(http.ListenAndServe(":8080", engine))
}
```
通过上述分析，`Web`服务的实现，主要是通过实现`ServeHttp`接口实现。
如下方式实现了一个静态的`web`框架.
```go
    package gee

    import (
       "fmt"
       "net/http"
    )

    type HandlerFunc func(http.ResponseWriter, *http.Request)

    type Engine struct {
       router map[string]HandlerFunc
    }

    func New() *Engine {
       return &Engine{router: make(map[string]HandlerFunc)}
    }

    func (engine *Engine) AddRoute(method string, pattern string, handler HandlerFunc) {
       key := method + "_" + pattern
       engine.router[key] = handler
    }

    func (engine *Engine) GET(pattern string, handler HandlerFunc) {
       engine.AddRoute("GET", pattern, handler)
    }

    func (engine *Engine) POST(pattern string, handler HandlerFunc) {
       engine.AddRoute("POST", pattern, handler)
    }

    func (engine *Engine) Run(addr string) (err error) {
       return http.ListenAndServe(addr, engine)
    }

    func (engine *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
       key := req.Method + "_" + req.URL.Path
       if handler, ok := engine.router[key]; ok {
          handler(w, req)
       } else {
          fmt.Fprintf(w, "404, Not FOUND!")
       }
    }
    
```
调用方式如下:
```go
/**
 * @program: GolangGee
 * @description:实现简单web服务器
 * @author: koritafei
 * @create: 2021-04-28 15:08
 * @version v0.1
 * */

package main

import (
   "fmt"
   "gee/gee"
   "net/http"
)

func main() {
   r := gee.New()
   r.GET("/", func(w http.ResponseWriter, req *http.Request){
      fmt.Fprintf(w, "req URL Path = %q\n", req.URL.Path)
   })
   
   r.GET("/hello", func(w http.ResponseWriter, req *http.Request){
      for key, val := range req.Header {
         fmt.Fprintf(w, "Header[%q] = %d\n", key, val)
      }
   })

   r.Run(":8080")
}
```
### 上下文
设计`context`的主要目的：
* 对`Web`服务来说，无非是根据`*http.Request`,构造响应的`http.ResponseWriter`。当这两者的接口太细，当需要构造一个完成的响应时，处理较为复杂。
* 封装相应的方法，简化接口调用。
* 支撑其他功能。
```go
package gee

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type H map[string]interface{}
type Context struct {
	// origin objects
	Writer http.ResponseWriter
	Req    *http.Request
	// request info
	Path   string
	Method string

	//response info
	StatusCode int
}

func newContext(w http.ResponseWriter, req *http.Request) *Context {
	return &Context{
		Writer: w,
		Req:    req,
		Path:   req.URL.Path,
		Method: req.Method,
	}
}

func (c *Context) PostForm(key string) string {
	return c.Req.FormValue(key)
}

func (c *Context) Query(key string) string {
	return c.Req.URL.Query().Get(key)
}

func (c *Context) Status(code int) {
	c.StatusCode = code
	c.Writer.WriteHeader(code)
}

func (c *Context) SetHeader(key string, value string) {
	c.Writer.Header().Set(key, value)
}

func (c *Context) String(code int, format string, value ...interface{}) {
	c.SetHeader("Context-Type", "text/plain")
	c.Status(code)
	c.Writer.Write([]byte(fmt.Sprintf(format, value)))
}

func (c *Context) JSON(code int, obj ...interface{}) {
	c.SetHeader("Context-Type", "application/json")
	c.Status(code)
	encoder := json.NewEncoder(c.Writer)
	if err := encoder.Encode(obj); err != nil {
		http.Error(c.Writer, err.Error(), 500)
	}
}

func (c *Context) Data(code int, data []byte) {
	c.Status(code)
	c.Writer.Write(data)
}

func (c *Context) HTML(code int, html string) {
	c.SetHeader("Context-Type", "text/html")
	c.Status(code)
	c.Writer.Write([]byte(html))
}
```
## 路由(`Router`)
```go
package gee

import (
	"log"
	"net/http"
)

type router struct {
	handlers map[string]HandlerFunc
}

func newRouter() *router {
	return &router{
		handlers: make(map[string]HandlerFunc),
	}
}

func (r *router) AddRouter(method string, pattern string, handler HandlerFunc) {
	log.Printf("Router %v - %v", method, pattern)
	key := method + "-" + pattern
	r.handlers[key] = handler
}

func (r *router) handle(c *Context) {
	key := c.Method + "-" + c.Path
	if handler, ok := r.handlers[key]; ok {
		handler(c)
	} else {
		c.String(http.StatusNotFound, "404 NOT FOUND, %s\n"), c.Path)
	}
}
```
## 框架入口
```go
package gee

import "net/http"

type HandlerFunc func(*Context)

type Engine struct {
	router *router
}

func New() *Engine {
	return &Engine{
		router: newRouter(),
	}
}

func (e *Engine) addRoute(method string, pattern string, handler HandlerFunc) {
	e.router.addRouter(method, pattern, handler)
}

func (e *Engine) GET(pattern string, handler HandlerFunc) {
	e.addRoute("GET", pattern, handler)
}

func (e *Engine) POST(pattern string, handler HandlerFunc) {
	e.addRoute("POST", pattern, handler)
}

func (e *Engine) Run(addr string) (err error) {
	return http.ListenAndServe(addr, e)
}

func (e *Engine) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c := newContext(w, r)
	e.router.handle(c)
}
```
## 前缀树路由
### `Trie`树简介
每个节点的所有子节点拥有相同的前缀。
`trie`树实现
```go
package gee

import "strings"

// trie 树实现

type node struct {
	pattern  string  // 待匹配路由，例如：/p/:lang
	part     string  // 路由中的一部分， 例如： :lang
	children []*node // 子节点，例如[doc, tutorial, intro]
	isWild   bool    // 是否精确匹配
}

// 第一个匹配成功的节点，用于插入
func (n *node) matchChild(part string) *node {
	for _, child := range n.children {
		if part == child.part || n.isWild {
			return child
		}
	}

	return nil
}

// 所有匹配成功的节点，用于匹配
func (n *node) matchChildren(part string) []*node {
	nodes := make([]*node, 0)
	for _, child := range n.children {
		if part == n.part {
			nodes = append(nodes, child)
		}
	}

	return nodes
}

func (n *node) insert(pattern string, parts []string, height int) {
	if len(parts) == height {
		n.pattern = pattern
		return
	}

	part := parts[height]
	child := n.matchChild(part)
	if nil == child {
		child = &node{
			part:   part,
			isWild: part[0] == ':' || part[1] == '*',
		}

		n.children = append(n.children, child)
	}
	child.insert(pattern, parts, height+1)
}

func (n *node) search(parts []string, height int) *node {
	if len(parts) == height || strings.HasPrefix(n.part, "*") {
		if "" == n.pattern {
			return nil
		}

		return n
	}

	part := parts[height]
	children := n.matchChildren(part)

	for _, child := range children {
		result := child.search(parts, height+1)
		if nil != result {
			return result
		}
	}
	return nil
}
```
## 分组
路由的分组控制。
分组是指将路由分组。对某一组路由进行相似的处理。如：
1. 以`/post`开头的路由匿名可访问；
2. 以`/admin`开头的路由需要鉴权；
3. 以`/api`开头的路由为`RESTful`接口，对接第三方，需要第三方鉴权。

大部分情况下，路由分组是通过相同的前缀来实现的。
每个分组可以使用同一个中间件，分组的子分组可拥有单独的中间件。
### 分组嵌套
一个`Group`对象的属性：
1. 前缀;要支持分组嵌套,需要知道当前分组的父亲是谁(`parent`).
```go
type HandlerFunc func(*Context)

type RouterGroup struct {
	prefix      string        // 前缀
	middlewares []HandlerFunc // middleware
	parent      *RouterGroup  // parent
	engine      *Engine
}
type Engine struct {
	*RouterGroup
	router *router
	groups []*RouterGroup
}

func New() *Engine {
	engine := &Engine{
		router: NewRouter(),
	}
	engine.RouterGroup = &RouterGroup{engine: engine}
	engine.groups = []*RouterGroup{engine.RouterGroup}
	return engine
}

func (group *RouterGroup) Group(prefix string) *RouterGroup {
	engine := group.engine
	newGroup := &RouterGroup{
		prefix: group.prefix + prefix,
		parent: group,
		engine: engine,
	}
	engine.groups = append(engine.groups, newGroup)
	return newGroup
}

func (e *Engine) Run(addr string) (err error) {
	return http.ListenAndServe(addr, e)
}

func (e *Engine) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c := newContext(w, r)
	e.router.handle(c)
}

func (group *RouterGroup) addRouter(method string, comp string, handler HandlerFunc) {
	pattern := group.prefix + comp
	log.Printf("Route %v-%v", method, pattern)
	group.engine.router.addRouter(method, pattern, handler)
}

func (group *RouterGroup) GET(pattern string, handler HandlerFunc) {
	group.addRouter("GET", pattern, handler)
}

func (group *RouterGroup) POST(pattern string, handler HandlerFunc) {
	group.addRouter("POST", pattern, handler)
}
```
## 中间件
非业务类技术组件.
提供一个对外接口，允许用户自定义相关功能，插入到框架。
关注的点：
1. 插入点在哪？
2. 中间件输入是什么？

`Gee`中间件设计：
1. 处理输入对象为`Context`;
2. 插入点为框架接受到请求初始化`Context`之后，允许用户使用定义的中间件做一些额外处理；
3. 通过调用`(*Context).Next()`函数，可以等待用户定义`handler`处理结束之后，做一些额外操作。
```go
func Logger() HandlerFunc {
	return func(c *Context) {
		// start timer
		t := time.Now()
		// process request
		c.Next()

		log.Printf("[%d] %s in %v", c.StatusCode, c.Req.RequestURI, time.Since(t))
	}
}
```
支持设置多个中间件，依次调用。
中间件是应用在`Group`之上，应用在最顶层的`Group`相当于作用全局。
`Context`的实现修改为：
```go
type Context struct {
	// origin objects
	Writer http.ResponseWriter
	Req    *http.Request
	// request info
	Path   string
	Method string
	Params map[string]string
	//response info
	StatusCode int

	handlers []HandlerFunc // 中间件
	index    int           // 中间件索引
}
```
其中`index`标记当前执行到第几个中间件，当调用`Next()`方法时，控制权交给下一个中间件,直到最后一个中间件，然后再从后向前,调用每个中间件在`Next`方法之后定义的操作。
```go
// 当接收到请求时，根据URL前缀判断适用的中间件
// 将中间件放入到handlers slice,依次调用
func (e *Engine) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var middlewares []HandlerFunc
	for _, group := range e.groups {
		if strings.HasPrefix(r.URL.Path, group.prefix) {
			middlewares = append(middlewares, group.middlewares...)
		}
	}
	c := newContext(w, r)
	c.handlers = middlewares
	e.router.handle(c)
}
```


