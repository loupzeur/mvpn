package vpn

import (
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

	lastMissIdx  byte //0 isn't an index :)
	lastMissTime time.Time
}

func NewVPN(iface *water.Interface, port int) VPNProcess {
	return VPNProcess{
		//Is the chan that get packet FROM the interface to internet (packet coming from INside)
		INChan: make(chan []byte),
		//is the chan that get packet FROM internet to the interface (packet coming from OUTside)
		OUTChan: make(chan []byte),
		Iface:   iface,
		Port:    port,
		//some rolling cache for resending packets in case of failure
		cacheIn:      &byteCache{Counter: 1, Data: map[int][]byte{}, LastCounter: time.Now()},
		cacheOut:     &byteCache{Counter: 1, Data: map[int][]byte{}, LastCounter: time.Now()},
		lastMissIdx:  0,
		lastMissTime: time.Now(),
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
	return data[0]
}

//ReturnOrderedData return the data in order until, if needed, the missing packet
func (s *byteCache) ReturnOrderedData() ([][]byte, []byte) {
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
		//!todo get average latency with ping and use it here
		if v > s.Counter+1 && time.Now().Before(s.LastCounter.Add(100*time.Millisecond)) {
			//need to be done in current latency * 2 (2 way for ask and answer)
			log.Printf("Missing packet %d\n", v)
			//ask the missing packet by asking it's index in the rolling cache
			s.LastCounter = s.LastCounter.Add(100 * time.Millisecond)
			last := v - 1
			if last == 0 {
				last = 255
			}
			return nil, []byte{byte(last)}
		}
		data = append(data, s.Data[v])
		//we need to keep data for a while, so we can resend it
		delete(s.Data, v)
		s.SetCounter(v)
	}
	return data, nil
}

//chanToIface channel data to the interface and reorder packets
func (s VPNProcess) chanToIface() {
	//we need to reorder packets in case both network are not in sync
	for data := range s.OUTChan {
		if len(data) == 1 {
			//this is a packet request, no need to pass it through the interface
			log.Printf("Sending Missing packet %d\n", data[0])
			if len(s.cacheIn.Data) > int(data[0]) {
				//data only contains the index, so we can append it's data
				//and send it back
				s.INChan <- append(data, s.cacheIn.Data[int(data[0])]...)
			}
			continue
		}
		s.cacheOut.Order(data)
		//we don't send everything, just elements that are in order
		data, missing := s.cacheOut.ReturnOrderedData()
		if missing != nil && (s.lastMissIdx != missing[0] || time.Now().After(s.lastMissTime.Add(100*time.Millisecond))) {
			log.Printf("Asking Missing packet %d\n", missing)
			s.INChan <- missing //send missing to the other side
			s.lastMissIdx = missing[0]
			s.lastMissTime = time.Now()
			continue
		}
		s.lastMissIdx = 0
		for _, v := range data {
			s.Iface.Write(v)
		}
		//log.Printf("Current Receiving counter %d len (%d)\n", s.cacheOut.Counter, len(s.cacheOut.Data))
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
		tmp := append(cCounter, packet[:plen]...)
		s.cacheIn.Order(tmp)
		s.INChan <- tmp
		//log.Printf("Current Sending counter %d\n", cCounter[0])
		cCounter[0]++ //will return to 0 once going above 255!
		if cCounter[0] == 0 {
			//0 is for a call for missing packet
			cCounter[0]++
		}
	}
}
