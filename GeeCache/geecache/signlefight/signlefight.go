package signlefight

import "sync"

// 代表正在进行，或已完成的请求
type call struct {
	wg sync.WaitGroup // 避免锁重入
	val interface{}
	err error
}

// 管理不同key的请求
type Group struct {
	mu sync.Mutex
	m map[string]*call
}

func (g *Group) Do(key string, fn func()(interface{}, error))(interface{}, error) {
	g.mu.Lock()
	if g.m == nil { // 延迟初始化
		g.m = make(map[string]*call)
	}

	if c, ok := g.m[key];ok {
		g.mu.Unlock()
		c.wg.Wait()
		return c.val, c.err
	}
	c := new(call)
	c.wg.Add(1)
	g.m[key] = c
	g.mu.Unlock()
	c.val,c.err = fn()
	c.wg.Done()
	g.mu.Lock()
	delete(g.m,key)
	g.mu.Unlock()
	return c.val, c.err
}
