package singleflight

// 官方准标准库有一个实现的  singleflight  // import "golang.org/x/sync/singleflight"

import "sync"

// 缓存雪崩：缓存在同一时刻全部失效，造成瞬时DB请求量大、压力骤增，引起雪崩。缓存雪崩通常因为缓存服务器宕机、缓存的 key 设置了相同的过期时间等引起。
// 缓存击穿：一个存在的key，在缓存过期的一刻，同时有大量的请求，这些请求都会击穿到 DB ，造成瞬时DB请求量大、压力骤增。
// 缓存穿透：查询一个不存在的数据，因为不存在则不会写到缓存中，所以每次都会去请求 DB，如果瞬间流量过大，穿透到 DB，导致宕机。

// call 代表正在进行中，或已经结束的请求。
type call struct {
	wg  sync.WaitGroup
	val interface{}
	err error
}

// Group 管理不同 key 的请求(call)。
type Group struct {
	mu sync.Mutex
	m  map[string]*call
}

// Do 针对相同的 key，无论 Do 被调用多少次，函数 fn 都只会被调用一次，等待 fn 调用结束了，返回返回值或错误。
func (g *Group) Do(key string, fn func() (interface{}, error)) (interface{}, error) {

	g.mu.Lock()
	if g.m == nil { // 延迟初始化 提高内存使用效率。
		g.m = make(map[string]*call)
	}

	if c, ok := g.m[key]; ok {
		g.mu.Unlock()
		c.wg.Wait() // 如果请求正在进行中 则等待
		return c.val, c.err
	}

	c := new(call)
	c.wg.Add(1)  // 发起请求前加锁
	g.m[key] = c //添加到 g.m，表明 key 已经有对应的请求在处理
	g.mu.Unlock()

	c.val, c.err = fn()
	c.wg.Done() //请求结束

	g.mu.Lock()
	delete(g.m, key)
	g.mu.Unlock()

	return c.val, c.err
}
