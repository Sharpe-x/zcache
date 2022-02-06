package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"
	"zcache"
)

var slowDB = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

func createGroup() *zcache.Group {
	return zcache.NewGroup("scores", 2<<10, zcache.GetterFunc(
		func(key string) ([]byte, error) {
			time.Sleep(time.Second * 3) //模拟耗时操作
			log.Println("[SlowDB] search key", key)
			if v, ok := slowDB[key]; ok {
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))
}

func startCacheServer(addr string, addRs []string, group *zcache.Group) { //启动缓存服务：创建 HTTPPool，添加节点信息
	peers := zcache.NewHTTPPool(addr)
	peers.Set(addRs...)
	group.RegisterPeers(peers)
	log.Println("zcache is running at", addr)
	log.Fatal(http.ListenAndServe(addr[7:], peers))
}

func startAPIServer(apiAddr string, group *zcache.Group) { // 用来启动一个 API 服务（端口 9999），与用户进行交互
	http.Handle("/api", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		key := req.URL.Query().Get("key")
		view, err := group.Get(key)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write(view.ByteSlice())
	}))
	log.Println("fontend server is running at", apiAddr)
	log.Fatal(http.ListenAndServe(apiAddr[7:], nil))
}

func main() {
	var port int
	var api bool
	flag.IntVar(&port, "port", 8000, "zcache server port")
	flag.BoolVar(&api, "api", false, "start a api server")
	flag.Parse()

	apiAddr := "http://127.0.0.1:9999"
	addrMap := map[int]string{
		8001: "http://127.0.0.1:8001",
		8002: "http://127.0.0.1:8002",
		8003: "http://127.0.0.1:8003",
	}

	var addRs []string
	for _, v := range addrMap {
		addRs = append(addRs, v)
	}

	zcacheGroup := createGroup()
	if api {
		go startAPIServer(apiAddr, zcacheGroup)
	}
	startCacheServer(addrMap[port], addRs, zcacheGroup)
}
