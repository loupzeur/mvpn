package vpn

import (
	"encoding/binary"
	"testing"
	"unsafe"
)

func TestByteOrder1(t *testing.T) {
	bc := byteCache{Counter: 0, Data: make(map[int][]byte)}
	bc.Order([]byte{1, 1, 1})
	bc.Order([]byte{2, 2, 1})
	bc.Order([]byte{3, 3, 1})
	bc.Order([]byte{5, 5, 1})

	data, _ := bc.ReturnOrderedData()
	for i, v := range data {
		t.Log(i, v)
	}
}

func TestByteOrder2(t *testing.T) {
	bc := &byteCache{Counter: 0, Data: make(map[int][]byte)}
	bc.Order([]byte{1, 1, 1})
	bc.Order([]byte{2, 2, 1})
	bc.Order([]byte{3, 3, 1})
	bc.Order([]byte{5, 5, 1})
	bc.Order([]byte{4, 4, 1})

	t.Log("all")
	data, _ := bc.ReturnOrderedData()
	for i, v := range data {
		t.Log(i, v)
	}
	t.Log("nothing")
	data, _ = bc.ReturnOrderedData()
	for i, v := range data {
		t.Log(i, v)
	}
	t.Log("4")
	data, _ = bc.ReturnOrderedData()
	for i, v := range data {
		t.Log(i, v)
	}
}

func TestByteOrder3(t *testing.T) {
	bc := &byteCache{Counter: 0, Data: make(map[int][]byte)}
	bc.Order([]byte{1, 1, 1})
	bc.Order([]byte{2, 2, 1})
	bc.Order([]byte{3, 3, 1})
	bc.Order([]byte{5, 5, 1})
	data, _ := bc.ReturnOrderedData()
	for i, v := range data {
		t.Log(i, v)
	}
	bc.Order([]byte{4, 4, 1})
	data, _ = bc.ReturnOrderedData()
	for i, v := range data {
		t.Log(i, v)
	}
}

//reset the cycle test
func TestByteOrder4(t *testing.T) {
	bc := &byteCache{Counter: 251, Data: make(map[int][]byte)}
	bc.Order([]byte{251, 1, 1})
	bc.Order([]byte{252, 2, 1})
	bc.Order([]byte{253, 3, 1})
	bc.Order([]byte{255, 5, 1})
	data, _ := bc.ReturnOrderedData()
	for i, v := range data {
		t.Log(i, v)
	}
	bc.Order([]byte{254, 4, 1})
	bc.Order([]byte{1, 6, 1})
	data, _ = bc.ReturnOrderedData()
	for i, v := range data {
		t.Log(i, v)
	}
	data, _ = bc.ReturnOrderedData()
	for i, v := range data {
		t.Log(i, v)
	}
}

func TestTntToByte(t *testing.T) {
	h := 65536
	i := (*[2]byte)(unsafe.Pointer(&h))[:]
	r := binary.LittleEndian.Uint16(i)

	//binary.LittleEndian.PutUint64()

	t.Log(h, i, r)
	t.Log(MaxCacheSize)

	t.Log(int(unsafe.Sizeof(0)))
}

func TestByteOverflow(t *testing.T) {
	b := []byte{255, 0}
	b[0]++
	t.Log(b)
	b[0]++
	t.Log(b)
	b[0]++
	t.Log(b)
}
