package zcache

// ByteView A ByteView holds an immutable view of the bytes
type ByteView struct {
	b []byte // 存储真实的缓存值
}

// Len returns the view's length
func (v ByteView) Len() int {
	return len(v.b)
}

// ByteSlice b 是只读的，使用 ByteSlice() 方法返回一个拷贝，防止缓存值被外部程序修改。
func (v ByteView) ByteSlice() []byte {
	return cloneBytes(v.b)
}

// cloneBytes 拷贝bytes 数组
func cloneBytes(b []byte) []byte {
	c := make([]byte, len(b))
	copy(c, b)
	return c
}

func (v ByteView) String() string {
	return string(v.b)
}
