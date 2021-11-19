package vpn

import (
	"github.com/songgao/water"
)

//Global stuff
var (
	MTU = 1440
)

//only needed once per server
type VPNProcess struct {
	Port    int
	INChan  chan []byte      //from interface to udp
	OUTChan chan []byte      //from udp to interface
	Iface   *water.Interface //interface for our server
}

func NewVPN(iface *water.Interface, port int) VPNProcess {
	return VPNProcess{INChan: make(chan []byte), OUTChan: make(chan []byte), Iface: iface, Port: port}
}
func (s VPNProcess) Run() {
	//one chan for data in
	go s.chanToIface()
	//one chan for data out
	go s.ifaceToChan()
}

//rolling cache on 1 byte (255 elements)
type byteCache struct {
	Counter byte
	Data    map[byte][]byte
}

//Order send data  in the right order
func (s *byteCache) Order(data []byte) byte {
	s.Data[data[0]] = data[1:]
	return data[0]
}

//ReturnOrderedData return the data in order until
func (s *byteCache) ReturnOrderedData() [][]byte {
	var data [][]byte
	for idx, v := range s.Data {
		if idx < s.Counter {
			continue
		}
		data = append(data, v)
		delete(s.Data, idx)
		s.Counter = idx
	}
	return data
}

//chanToIface channel data to the interface and reorder packets
func (s VPNProcess) chanToIface() {
	pReorderCache := byteCache{}
	last := byte(0)
	//we need to reorder packets in case both network are not in sync
	for data := range s.OUTChan {
		curIndex := pReorderCache.Order(data)
		if curIndex+1 > last+1 {
			continue
		}
		//we don't send everything, just elements that are in order
		for _, v := range pReorderCache.ReturnOrderedData() {
			s.Iface.Write(v)
		}
	}
}

//ifaceToChan channel data to the interface and count packets
func (s VPNProcess) ifaceToChan() {
	packet := make([]byte, MTU+1)
	cCounter := []byte{0, 0}
	for {
		plen, err := s.Iface.Read(packet)
		if err != nil {
			break
		}
		s.INChan <- append(cCounter[:1], packet[:plen]...)
		cCounter[0] = (cCounter[0] + 1) % 255 //will never overflow
	}
}
