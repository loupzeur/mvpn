package vpn

import (
	"fmt"
	"sort"
	"time"

	"github.com/songgao/water"
)

//Global stuff
var (
	MTU          = 1440
	MaxCacheSize = 256
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

//rolling cache on 1 byte (MaxCacheSize elements)
type byteCache struct {
	Counter     int
	Data        map[int][]byte
	LastCounter time.Time
}

func (s *byteCache) SetCounter(counter int) {
	s.Counter = counter
	s.LastCounter = time.Now()
}

//Order send data  in the right order
func (s *byteCache) Order(data []byte) byte {
	s.Data[int(data[0])] = data[1:]
	fmt.Printf("Adding counter : %d\n", data[0])
	return data[0]
}

//ReturnOrderedData return the data in order until
func (s *byteCache) ReturnOrderedData() [][]byte {
	var data [][]byte
	keys := []int{}
	for i := range s.Data {
		keys = append(keys, i)
	}
	sort.Ints(keys)
	for _, v := range keys {
		if v < s.Counter%MaxCacheSize {
			continue
		}
		//add a timeout to send anyway if no packets is received
		//need a way to reask not received packet
		if v > s.Counter+1 && time.Now().Before(s.LastCounter.Add(1*time.Second)) {
			//todo ask to resend packet here and wait answer
			//need to be done in current latency * 4 (2 way for ask and answer)
			break
		}
		data = append(data, s.Data[v])
		delete(s.Data, v)
		s.SetCounter(v)
	}
	return data
}

//chanToIface channel data to the interface and reorder packets
func (s VPNProcess) chanToIface() {
	pReorderCache := &byteCache{Counter: 0, Data: map[int][]byte{}, LastCounter: time.Now()}
	//we need to reorder packets in case both network are not in sync
	for data := range s.OUTChan {
		pReorderCache.Order(data)
		//we don't send everything, just elements that are in order
		for _, v := range pReorderCache.ReturnOrderedData() {
			s.Iface.Write(v)
		}
		fmt.Printf("Current Receiving counter %d len (%d)\n", pReorderCache.Counter, len(pReorderCache.Data))
	}
}

//ifaceToChan channel data to the interface and count packets
func (s VPNProcess) ifaceToChan() {
	packet := make([]byte, MTU+1)
	cCounter := []byte{0}
	for {
		plen, err := s.Iface.Read(packet)
		if err != nil {
			break
		}
		s.INChan <- append(cCounter, packet[:plen]...)
		fmt.Printf("Current Sending counter %d\n", cCounter[0])
		cCounter[0]++ //will return to 0 once going above 255!
	}
}
