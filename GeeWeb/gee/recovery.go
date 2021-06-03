package gee

import (
	"fmt"
	"log"
	"runtime"
	"strings"
)

func Recovery() HandlerFunc {
	return func(c *Context) {
		defer func() {
			if err := recover(); err != nil {
				message := fmt.Sprintf("%s", err)
				log.Printf("%s\n\n", trace(message))
				log.Printf("Internal Server Error")
			}
		}()

		c.Next()
	}

}

func trace(msg string) string {
	var pcs [32]uintptr
	n := runtime.Callers(3, pcs[:])
	var str strings.Builder
	str.WriteString(msg + "\nTraceback:")
	for _, pc := range pcs[:n] {
		fn := runtime.FuncForPC(pc)   // 获取对应的函数
		file, line := fn.FileLine(pc) // 获取函数所在的文件名和行号
		str.WriteString(fmt.Sprintf("\n%s\t%s\n", file, line))
	}

	return str.String()
}
