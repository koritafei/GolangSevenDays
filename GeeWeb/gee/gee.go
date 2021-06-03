package gee

import (
	"html/template"
	"log"
	"net/http"
	"path"
	"strings"
)

type HandlerFunc func(*Context)

type RouterGroup struct {
	prefix      string        // 前缀
	middlewares []HandlerFunc // middleware
	parent      *RouterGroup  // parent
	engine      *Engine
}
type Engine struct {
	*RouterGroup
	router       *router
	groups       []*RouterGroup
	htmlTemplate *template.Template
	funcMap      template.FuncMap
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

/*
func (e *Engine) addRoute(method string, pattern string, handler HandlerFunc) {
	e.router.addRouter(method, pattern, handler)
}

func (e *Engine) GET(pattern string, handler HandlerFunc) {
	e.addRoute("GET", pattern, handler)
}

func (e *Engine) POST(pattern string, handler HandlerFunc) {
	e.addRoute("POST", pattern, handler)
}
*/

func (e *Engine) Run(addr string) (err error) {
	return http.ListenAndServe(addr, e)
}

// func (e *Engine) ServeHTTP(w http.ResponseWriter, r *http.Request) {
// 	c := newContext(w, r)
// 	e.router.handle(c)
// }

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

func (group *RouterGroup) Use(middlerwares ...HandlerFunc) {
	group.middlewares = append(group.middlewares, middlerwares...)
}

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
	c.engine = e
	e.router.handle(c)
}

func (group *RouterGroup) createStaticHandler(relativePath string, fs http.FileSystem) HandlerFunc {
	absolutePath := path.Join(group.prefix, relativePath)
	fileServer := http.StripPrefix(absolutePath, http.FileServer(fs))
	return func(c *Context) {
		file := c.Param("filepath")
		if _, err := fs.Open(file); err != nil {
			c.Status(http.StatusNotFound)
			return
		}

		fileServer.ServeHTTP(c.Writer, c.Req)
	}
}

func (group *RouterGroup) Static(relativePath string, root string) {
	handler := group.createStaticHandler(relativePath, http.Dir(root))
	urlPattern := path.Join(relativePath, "/*filePath")
	group.GET(urlPattern, handler)
}

func (engine *Engine) SetFuncMap(funcMap template.FuncMap) {
	engine.funcMap = funcMap
}

func (engine *Engine) LoadHtmlGlob(pattern string) {
	engine.htmlTemplate = template.Must(template.New("").Funcs(engine.funcMap).ParseGlob(pattern))
}
