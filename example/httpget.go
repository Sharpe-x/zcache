package main

import (
	"fmt"
	"log"
	"net/http"
	"zcache"
)

var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

func getFromDb(key string) ([]byte, error) {
	log.Println("[SlowDB] search key", key)
	if v, ok := db[key]; ok {
		return []byte(v), nil
	}
	return nil, fmt.Errorf("%s not exist", key)
}

// curl http://127.0.0.1:9999/_zcache/scores/Tom
func main() {
	zcache.NewGroup("scores", 2<<10, zcache.GetterFunc(getFromDb))
	log.Println("zcache is running at 127.0.0.1:9999")
	log.Fatal(http.ListenAndServe(":9999", zcache.NewHTTPPool(":9999")))
}
