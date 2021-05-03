package main

import (
	"fmt"
	"log"
	"net"
	"sync"

	"github.com/songgao/water"
	//https://github.com/buger/goterm
)

//Global stuff
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
func (s VPNProcess) chanToIface() {
	for data := range s.OUTChan {
		s.Iface.Write(data)
	}
}
func (s VPNProcess) ifaceToChan() {
	packet := make([]byte, *MTU)
	for {
		plen, err := s.Iface.Read(packet)
		if err != nil {
			break
		}
		s.INChan <- packet[:plen]
	}
}

//ProcessConnection process server stuff
func (s *VPNProcess) ProcessConnection() {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%v", s.Port))
	if nil != err {
		log.Fatalln("Unable to get UDP socket:", err)
	}
	conn, err := net.ListenUDP("udp", addr)
	if nil != err {
		log.Fatalln("Unable to listen on UDP socket:", err)
	}
	var last *net.UDPAddr
	go func() {
		buf := make([]byte, *MTU)
		for {
			n, addr, err := conn.ReadFromUDP(buf)
			if err != nil || n == 0 {
				fmt.Println("Error: ", err)
				continue
			}
			last = addr
			s.OUTChan <- buf[:n]
		}
	}()
	for data := range s.INChan {
		if last != nil {
			conn.WriteToUDP(data, last)
		}
	}
}

//ProcessClient process some client stuff
func (s *VPNProcess) ProcessClient(lip *net.UDPAddr, rip *net.UDPAddr, wg *sync.WaitGroup) {
	defer wg.Done()
	conn, err := net.ListenUDP("udp", lip)
	if err != nil {
		log.Fatalln("Unable to get UDP socket:", err)
	}
	log.Println("Listening on :", lip.String())
	go func() {
		packet := make([]byte, *MTU)
		for {
			plen, err := conn.Read(packet)
			if err != nil {
				break
			}
			s.OUTChan <- packet[:plen]
		}
	}()
	for data := range s.INChan {
		n, err := conn.WriteToUDP(data, rip)
		if err != nil {
			log.Println("Error writing packet ", n, err)
		}
	}
}
