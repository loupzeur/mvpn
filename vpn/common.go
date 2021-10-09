package vpn

import (
	"log"

	"github.com/songgao/water"
	"golang.org/x/net/ipv4"
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

//func that takes array of byte of ip header and return sequence number
//return src, dst, seq and ack from tcp packet
func getSequence(data []byte) (int, int, int, int) {
	startTcp := 0 //int(data[0]&0x0f) << 2 //length of ip header from packet
	lenH := len(data)

	i := startTcp + 4 //start of sequence
	src := int(data[0])<<8 | int(data[1])
	dst := int(data[2])<<8 | int(data[3])
	if i+7 > lenH {
		return src, dst, 0, 0
	}
	return src, dst,
		int(data[(i)])<<24 | int(data[i+1])<<16 | int(data[i+2])<<8 | int(data[i+3]),
		int(data[(i+4)])<<24 | int(data[i+5])<<16 | int(data[i+6])<<8 | int(data[i+7])
}

func (s VPNProcess) chanToIface() {
	for data := range s.OUTChan {
		//need to reorder tcp packet going out of interface
		if len(data) > 12 && data[9] == 0x06 {
			h, _ := ipv4.ParseHeader(data)
			src, dst, seq, ack := getSequence(data[h.Len:])
			log.Printf("<=%s %s %d %d %d %d %d\n", h.Dst.String(), h.Src.String(), h.Protocol, src, dst, seq, ack)
			//will require reordering of the packet
		}
		s.Iface.Write(data)
	}
}
func (s VPNProcess) ifaceToChan() {
	packet := make([]byte, MTU)
	for {
		plen, err := s.Iface.Read(packet)
		if err != nil {
			break
		}
		s.INChan <- packet[:plen]
	}
}
