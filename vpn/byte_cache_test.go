package vpn

import "testing"

func TestByteOrder1(t *testing.T) {
	bc := byteCache{Counter: 0, Data: make(map[byte][]byte)}
	bc.Order([]byte{1, 1, 1})
	bc.Order([]byte{2, 1, 1})
	bc.Order([]byte{3, 1, 1})
	bc.Order([]byte{5, 1, 1})

	for i, v := range bc.ReturnOrderedData() {
		t.Log(i, v)
	}
}

func TestByteOrder2(t *testing.T) {
	bc := &byteCache{Counter: 0, Data: make(map[byte][]byte)}
	bc.Order([]byte{1, 1, 1})
	bc.Order([]byte{2, 1, 1})
	bc.Order([]byte{3, 1, 1})
	bc.Order([]byte{5, 1, 1})
	bc.Order([]byte{4, 1, 1})

	t.Log("all")
	for i, v := range bc.ReturnOrderedData() {
		t.Log(i, v)
	}
	t.Log("nothing")
	for i, v := range bc.ReturnOrderedData() {
		t.Log(i, v)
	}
	t.Log("4")
	for i, v := range bc.ReturnOrderedData() {
		t.Log(i, v)
	}
}

func TestByteOrder3(t *testing.T) {
	bc := &byteCache{Counter: 0, Data: make(map[byte][]byte)}
	bc.Order([]byte{1, 1, 1})
	bc.Order([]byte{2, 1, 1})
	bc.Order([]byte{3, 1, 1})
	bc.Order([]byte{5, 1, 1})

	for i, v := range bc.ReturnOrderedData() {
		t.Log(i, v)
	}
	bc.Order([]byte{4, 1, 1})
	for i, v := range bc.ReturnOrderedData() {
		t.Log(i, v)
	}
}
