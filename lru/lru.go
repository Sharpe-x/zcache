package lru

import "container/list"

// FIFO/LFU/LRU 算法简介
// FIFO(First In First Out) 先进先出，也就是淘汰缓存中最老(最早添加)的记录。 但是很多场景下，部分记录虽然是最早添加但也最常被访问，而不得不因为呆的时间太长而被淘汰。这类数据会被频繁地添加进缓存，又被淘汰出去，导致缓存命中率降低。
// LFU(Least Frequently Used) 最少使用，也就是淘汰缓存中访问频率最低的记录。
// 最近最少使用，相对于仅考虑时间因素的 FIFO 和仅考虑访问频率的 LFU，LRU 算法可以认为是相对平衡的一种淘汰算法。LRU 认为，如果数据最近被访问过，
// 那么将来被访问的概率也会更高。LRU 算法的实现非常简单，维护一个队列，如果某条记录被访问了，则移动到队尾，那么队首则是最近最少访问的数据，淘汰该条记录即可。

type Cache struct {
	maxBytes int64                    //允许使用的最大内存
	nBytes   int64                    // 是当前已使用的内存
	ll       *list.List               //  Go 语言标准库实现的双向链表list.List。
	cache    map[string]*list.Element // 字典 键是字符串，值是双向链表中对应节点的指针。

	// optional and executed when an entry is purged.
	OnEvicted func(key string, value Value) // 是某条记录被移除时的回调函数，可以为 nil。
}

//  entry 双向链表节点的数据类型
type entry struct {
	key   string
	value Value
}

// Value use Len to count how many bytes it takes 返回值所占用的内存大小。
type Value interface {
	Len() int
}

func New(maxBytes int64, onEvicted func(string, Value)) *Cache {
	return &Cache{
		maxBytes:  maxBytes,
		ll:        list.New(),
		cache:     make(map[string]*list.Element),
		OnEvicted: onEvicted,
	}
}

// Get 查找key 对应的value
func (c *Cache) Get(key string) (Value, bool) {
	if elem, ok := c.cache[key]; ok { //如果键对应的链表节点存在，则将对应节点移动到队尾，并返回查找到的值。
		c.ll.MoveToFront(elem) // 双向链表作为队列，队首队尾是相对的，在这里约定 front 为队尾
		kv := elem.Value.(*entry)
		return kv.value, true
	}
	return nil, false
}

// RemoveOldest 缓存淘汰 最近最少访问
func (c *Cache) RemoveOldest() {
	ele := c.ll.Back() // 取到队首节点
	if ele != nil {
		c.ll.Remove(ele) // ，从链表中删除。
		kv := ele.Value.(*entry)
		delete(c.cache, kv.key)                                // 从字典中 c.cache 删除该节点的映射关系。
		c.nBytes -= int64(len(kv.key)) + int64(kv.value.Len()) // 更新当前所用的内存
		if c.OnEvicted != nil {
			c.OnEvicted(kv.key, kv.value) // 执行回调函数
		}
	}
}

// Add 新增/修改
func (c *Cache) Add(key string, value Value) {
	if ele, ok := c.cache[key]; ok { // 如果键存在，则更新对应节点的值，并将该节点移到队尾。
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		c.nBytes += int64(value.Len()) - int64(kv.value.Len())
		kv.value = value
	} else { // 不存在则是新增场景，首先队尾添加新节点 &entry{key, value}, 并字典中添加 key 和节点的映射关系。
		ele = c.ll.PushFront(&entry{key: key, value: value})
		c.cache[key] = ele
		c.nBytes += int64(len(key)) + int64(value.Len())
	}

	// 如果超过了设定的最大值 c.maxBytes，则移除最少访问的节点。
	for c.maxBytes != 0 && c.maxBytes < c.nBytes {
		c.RemoveOldest()
	}
}

// Len the number of cache entries
func (c *Cache) Len() int {
	return c.ll.Len()
}
