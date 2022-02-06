// Package zcache http.go 提供被其他节点访问的能力(基于http)
package zcache

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"zcache/consistenthash"
)

// 分布式缓存需要实现节点间通信，建立基于 HTTP 的通信机制是比较常见和简单的做法。
// 如果一个节点启动了 HTTP 服务，那么这个节点就可以被其他节点访问。

const (
	defaultReplicas = 50
	defaultBasePath = "/_zcache/" // 节点间通讯地址的前缀
)

type HTTPPool struct {
	self        string                 // 记录自己的地址，包括主机名/IP 和端口
	basePath    string                 // 节点间通讯地址的前缀
	mu          sync.Mutex             // guards peers and httpGetters
	peers       *consistenthash.Map    // 一致性哈希算法的 Map，用来根据具体的 key 选择节点
	httpGetters map[string]*httpGetter // 映射远程节点与对应的 httpGetter。
}

func NewHTTPPool(self string) *HTTPPool {
	return &HTTPPool{
		self:     self,
		basePath: defaultBasePath,
	}
}

// Log info with server name
func (h *HTTPPool) Log(format string, v ...interface{}) {
	log.Printf("[Server %s] %s", h.self, fmt.Sprintf(format, v...))
}

func (h *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, h.basePath) {
		panic("HttpPool serving unexpected path: " + r.URL.Path)
	}

	h.Log("%s %s", r.Method, r.URL.Path)
	// /<basepath>/<groupname>/<key> 约定的访问路径
	parts := strings.SplitN(r.URL.Path[len(h.basePath):], "/", 2)
	if len(parts) != 2 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	groupName := parts[0]
	key := parts[1]
	group := GetGroup(groupName)
	if group == nil {
		http.Error(w, "no such group: "+groupName, http.StatusNotFound)
		return
	}

	view, err := group.Get(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	_, _ = w.Write(view.ByteSlice())
}

// 确保struct httpGetter 实现了接口 PeerGetter
var _ PeerGetter = (*httpGetter)(nil)

type httpGetter struct {
	baseURL string // 将要访问的远程节点的地址
}

// Get 获取返回值，并转换为 []bytes 类型
func (h *httpGetter) Get(group string, key string) ([]byte, error) {
	getUrl := fmt.Sprintf(
		"%v%v/%v",
		h.baseURL,
		url.QueryEscape(group),
		url.QueryEscape(key),
	)
	log.Println("http.Get url:", getUrl)
	res, err := http.Get(getUrl)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned:%v", res.Status)
	}

	bytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %v", err)
	}

	return bytes, nil
}

// 确保struct HTTPPool 实现了接口 PeerPicker
var _ PeerPicker = (*HTTPPool)(nil)

// PickPeer 包装了一致性哈希算法的 Get() 方法，根据具体的 key，选择节点，返回节点对应的 HTTP 客户端。
func (h *HTTPPool) PickPeer(key string) (peer PeerGetter, ok bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	// peer 是远程节点不是自己
	if peer := h.peers.Get(key); peer != "" && peer != h.self {
		h.Log("Pick peer %s", peer)
		return h.httpGetters[peer], true
	}
	return nil, false
}

// Set 实例化一致性哈希算法，并且添加了传入的节点。
func (h *HTTPPool) Set(peers ...string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.peers = consistenthash.New(defaultReplicas, nil)
	h.peers.Add(peers...)
	h.httpGetters = make(map[string]*httpGetter, len(peers))
	for _, peer := range peers {
		h.httpGetters[peer] = &httpGetter{
			baseURL: peer + h.basePath,
		}
	}
}
