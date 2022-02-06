package zcache

import "zcache/api"

// PeerPicker is the interface that must be implemented to locate
// the peer that owns a specific key.
type PeerPicker interface {
	PickPeer(key string) (peer PeerGetter, ok bool) // 用于根据传入的 key 选择相应节点 PeerGetter。
}

// PeerGetter is the interface that must be implemented by a peer.
type PeerGetter interface {
	// Get (group string, key string) ([]byte, error) // 用于从对应 group 查找缓存值。
	Get(request *api.Request, response *api.Response) error
}
