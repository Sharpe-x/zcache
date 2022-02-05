package zcache

import (
	"fmt"
	"log"
	"reflect"
	"testing"
	"time"
)

var db = map[string]string{
	"Tom":  "630",
	"Jack": "650",
	"Sam":  "567",
}

func TestGetter(t *testing.T) {
	var f Getter = GetterFunc(func(key string) ([]byte, error) {
		return []byte(key), nil
	})

	f2 := GetterFunc(func(key string) ([]byte, error) {
		return []byte(key), nil
	})

	expect := []byte("key")
	if v, _ := f.Get("key"); !reflect.DeepEqual(v, expect) {
		t.Errorf("callback failed")
	}

	if v, _ := f2.Get("key"); !reflect.DeepEqual(v, expect) {
		t.Errorf("callback failed")
	}
}

func TestGet(t *testing.T) {
	loadCounts := make(map[string]int, len(db))
	zcache := NewGroup("scores", 2<<10, GetterFunc(func(key string) ([]byte, error) {

		log.Println("[SlowDb] search key ", key)
		if v, ok := db[key]; ok {
			if _, ok := loadCounts[key]; !ok {
				loadCounts[key] = 0
			}
			loadCounts[key] += 1
			return []byte(v), nil
		}
		return nil, fmt.Errorf("%s not exist", key)
	}))

	for k, v := range db {
		if view, err := zcache.Get(k); err != nil || view.String() != v {
			t.Fatal("failed to get value of Tom")
		}

		if _, err := zcache.Get(k); err != nil || loadCounts[k] > 1 {
			t.Fatalf("cache %s miss", k)
		}
	}

	if view, err := zcache.Get("unknown"); err == nil {
		t.Fatalf("the value of unknow should be empty, but %s got", view)
	}

}

func GetFromUserSource(key string) ([]byte, error) {

	log.Println("[GetFromUserSource] search key ", key)
	time.Sleep(time.Second) // 模拟从自定义源获取数据

	if key == "unknown" {
		return nil, fmt.Errorf("%s not exist", key)
	}

	return []byte(key + "_value"), nil
}

func TestGetterFunc(t *testing.T) {
	// GetterFunc(GetFromUserSource) 将普通函数GetFromUserSource 转化为Getter 接口类型
	zcache := NewGroup("scores", 2<<10, GetterFunc(GetFromUserSource))

	for k, _ := range db {
		if view, err := zcache.Get(k); err != nil || view.String() != k+"_value" {
			t.Fatal("failed to get value of Tom")
		}

		if _, err := zcache.Get(k); err != nil {
			t.Fatalf("cache %s miss", k)
		}
	}

	if view, err := zcache.Get("unknown"); err == nil {
		t.Fatalf("the value of unknow should be empty, but %s got", view)
	}

}
