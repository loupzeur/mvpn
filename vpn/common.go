package vpn

import (
	"fmt"
	"log"
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

	cacheIn  *byteCache
	cacheOut *byteCache
}

func NewVPN(iface *water.Interface, port int) VPNProcess {
	return VPNProcess{
		INChan:   make(chan []byte),
		OUTChan:  make(chan []byte),
		Iface:    iface,
		Port:     port,
		cacheIn:  &byteCache{Counter: 1, Data: map[int][]byte{}, LastCounter: time.Now()},
		cacheOut: &byteCache{Counter: 1, Data: map[int][]byte{}, LastCounter: time.Now()},
	}
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

//ReturnOrderedData return the data in order until, if needed, the missing packet
func (s *byteCache) ReturnOrderedData() ([][]byte, [][]byte) {
	var data [][]byte
	var missing [][]byte
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
		//!todo get average latency with ping and use it here
		if v > s.Counter+1 && time.Now().Before(s.LastCounter.Add(1*time.Second)) {
			//need to be done in current latency * 2 (2 way for ask and answer)
			log.Printf("Missing packet %d\n", v)
			//ask the missing packet by asking it's index in the rolling cache
			missing = append(missing, []byte{byte(v)})
			break
		}
		data = append(data, s.Data[v])
		delete(s.Data, v)
		s.SetCounter(v)
	}
	return data, missing
}

//chanToIface channel data to the interface and reorder packets
func (s VPNProcess) chanToIface() {
	//we need to reorder packets in case both network are not in sync
	for data := range s.OUTChan {
		if len(data) == 1 {
			//this is a packet request, no need to pass it through the interface
			log.Printf("Sending Missing packet %d\n", data[0])
			if len(s.cacheIn.Data) < int(data[0]) {
				//data only contains the index, so we can append it's data
				//and send it back
				s.INChan <- append(data, s.cacheIn.Data[int(data[0])]...)
			}
			continue
		}
		s.cacheOut.Order(data)
		//we don't send everything, just elements that are in order
		data, missing := s.cacheOut.ReturnOrderedData()
		for _, v := range missing {
			s.INChan <- v //send missing to the other side
		}
		for _, v := range data {
			s.Iface.Write(v)
		}
		fmt.Printf("Current Receiving counter %d len (%d)\n", s.cacheOut.Counter, len(s.cacheOut.Data))
	}
}

//ifaceToChan channel data to the interface and count packets
func (s VPNProcess) ifaceToChan() {
	packet := make([]byte, MTU+1)
	cCounter := []byte{1}
	for {
		plen, err := s.Iface.Read(packet)
		if err != nil {
			break
		}
		//cache for outgoing packets
		s.cacheIn.Order(packet[:plen])
		s.INChan <- append(cCounter, packet[:plen]...)
		fmt.Printf("Current Sending counter %d\n", cCounter[0])
		cCounter[0]++ //will return to 0 once going above 255!
		if cCounter[0] == 0 {
			//0 is for a call for missing packet
			cCounter[0]++
		}
	}
}
