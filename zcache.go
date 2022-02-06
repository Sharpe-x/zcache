package zcache

import (
	"fmt"
	xSingleflight "golang.org/x/sync/singleflight"
	"log"
	"sync"
	"zcache/singleflight"
)

// 接受key  检查是否被缓存  是 返回缓存值
//                       否  是否应该从远程节点获取  是  与远程节点交互 返回缓存值
//                                               否   调用回调函数 获取值并添加到缓存中 返回缓存值

// 如果缓存不存在 应该从数据源 获取数据并添加到缓存中
// 框架不应该支持多种数据源的配置 因为数据源种类太多 没法一一实现 而是拓展性不好
// 这件事交给用户来做 框架提供回调函数的模型 用户自己实现

// A Getter loads data for a key.
type Getter interface { // 定义接口 Getter 和 回调函数 Get(key string)([]byte, error)，参数是 key，返回值是 []byte
	Get(key string) ([]byte, error)
}

// 函数类型实现某一个接口，称之为接口型函数，方便使用者在调用时既能够传入函数作为参数，也能够传入实现了该接口的结构体作为参数。
// 接口型函数只能应用于接口内部只定义了一个方法的情况
// https://geektutu.com/post/7days-golang-q1.html

// A GetterFunc implements Getter with a function
type GetterFunc func(key string) ([]byte, error) // 定义函数类型 GetterFunc，并实现 Getter 接口的 Get 方法。

// Get implements Getter interface function
func (f GetterFunc) Get(key string) ([]byte, error) { // 定义一个函数类型 F，并且实现接口 A 的方法，
	// 然后在这个方法中调用自己。这是 Go 语言中将其他函数（参数返回值定义与 F 一致）转换为接口 A 的常用技巧。
	// 如果不提供这个把函数转换为接口的函数，你调用时就需要创建一个struct，然后实现对应的接口，创建一个实例作为参数
	return f(key)
}

type Group struct {
	name      string               // 唯一的名称 name
	getter    Getter               // 缓存未命中时获取源数据的回调(callback)
	maniCache cache                // 并发缓存
	peers     PeerPicker           // 节点选择
	loader    *singleflight.Group  // 防止缓存击穿  singleflight 实现
	xLoader   *xSingleflight.Group // 准官方库golang.org/x/sync/singleflight
}

var (
	mu     sync.RWMutex
	groups = make(map[string]*Group)
)

// NewGroup create a new instance of Group
func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	if getter == nil {
		panic("nil Getter")
	}
	mu.Lock()
	defer mu.Unlock()

	g := &Group{
		name:      name,
		getter:    getter,
		maniCache: cache{cacheBytes: cacheBytes},
		loader:    &singleflight.Group{},
		xLoader:   &xSingleflight.Group{},
	}
	groups[name] = g
	return g
}

func GetGroup(name string) *Group {
	mu.RLock()
	g := groups[name]
	mu.RUnlock()
	return g
}

func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("key is required")
	}

	if v, ok := g.maniCache.get(key); ok { // 从 mainCache 中查找缓存，如果存在则返回缓存值。
		log.Println("[zcache] hit")
		return v, nil
	}

	return g.load(key) //  缓存不存在，则调用 load 方法，load 调用 getLocally
	// getLocally 调用用户回调函数 g.getter.Get() 获取源数据，
	//并且将源数据添加到缓存 mainCache 中（通过 populateCache 方法）

}

func (g *Group) load(key string) (value ByteView, err error) {

	//view, err := g.loader.Do(key, func() (interface{}, error) {
	view, err, _ := g.xLoader.Do(key, func() (interface{}, error) {
		// 选择节点，若非本机节点   调用 getFromPeer() 从远程获取。
		if g.peers != nil {
			if peer, ok := g.peers.PickPeer(key); ok {
				log.Println("[zcache] get from peer:", peer)
				if value, err = g.getFromPeer(peer, key); err == nil {
					return value, nil
				}
				log.Println("[zcache] Failed to get from peer", err)
			}
		}
		// 回退到getLocally
		return g.getLocally(key)
	})

	if err == nil {
		return view.(ByteView), nil
	}

	return
}

func (g *Group) getLocally(key string) (ByteView, error) {
	bytes, err := g.getter.Get(key)
	if err != nil {
		return ByteView{}, err
	}

	value := ByteView{
		b: cloneBytes(bytes),
	}

	g.populateCache(key, value)
	return value, nil
}

func (g *Group) populateCache(key string, value ByteView) {
	g.maniCache.add(key, value)
}

//RegisterPeers  将实现了PeerPicker 接口的 HTTPPool 注入到 Group
func (g *Group) RegisterPeers(peers PeerPicker) {
	if g.peers != nil {
		panic("RegisterPeerPicker called more than once")
	}
	g.peers = peers
}

// 访问远程节点，获取缓存值。
func (g *Group) getFromPeer(peer PeerGetter, key string) (ByteView, error) {
	bytes, err := peer.Get(g.name, key)
	if err != nil {
		return ByteView{}, err
	}

	return ByteView{
		b: bytes,
	}, nil
}
