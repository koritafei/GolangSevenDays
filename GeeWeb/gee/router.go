package gee

import (
	"net/http"
	"strings"
)

type router struct {
	roots    map[string]*node
	handlers map[string]HandlerFunc
}

func NewRouter() *router {
	return &router{
		roots:    make(map[string]*node),
		handlers: make(map[string]HandlerFunc),
	}
}

func parsePattern(pattern string) []string {
	vs := strings.Split(pattern, "/")
	parts := make([]string, 0)

	for _, item := range vs {
		if len(item) != 0 {
			parts = append(parts, item)
			if item[0] == '*' {
				break
			}
		}
	}

	return parts
}

func (r *router) addRouter(method string, pattern string, handler HandlerFunc) {
	parts := parsePattern(pattern)
	key := method + "-" + pattern
	_, ok := r.roots[method]
	if !ok {
		r.roots[method] = &node{}
	}

	r.roots[method].insert(pattern, parts, 0)
	r.handlers[key] = handler
}

func (r *router) getRouter(method string, pattern string) (*node, map[string]string) {
	searchParts := parsePattern(pattern)
	params := make(map[string]string)
	root, ok := r.roots[method]
	if !ok {
		return nil, nil
	}

	n := root.search(searchParts, 0)

	if n != nil {
		parts := parsePattern(n.pattern)
		for index, part := range parts {
			if ':' == part[0] {
				params[part[1:]] = searchParts[index]
			}
			if '*' == part[0] && len(part) > 1 {
				params[part[1:]] = strings.Join(searchParts[index:], "/")
				break
			}
		}

		return n, params
	}

	return nil, nil
}

// func (r *router) handle(c *Context) {
// 	n, params := r.getRouter(c.Method, c.Path)
// 	if nil != n {
// 		c.Params = params
// 		key := c.Method + "-" + n.pattern
// 		r.handlers[key](c)
// 	} else {
// 		c.String(http.StatusNotFound, "404 Not Found, %s\n", c.Path)
// 	}
// }

func (r *router) handle(c *Context) {
	n, params := r.getRouter(c.Method, c.Path)
	if nil != n {
		key := c.Method + "-" + n.pattern
		c.Params = params
		c.handlers = append(c.handlers, r.handlers[key])
	} else {
		c.handlers = append(c.handlers, func(c *Context) {
			c.String(http.StatusNotFound, "404, Not Found, %s\n", c.Path)
		})
	}
	c.Next()
}
