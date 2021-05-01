package main

import (
	"fmt"
	"log"
	"net"

	"github.com/songgao/water"
	"golang.org/x/net/ipv4"
)

//Global stuff
//only needed once per server
type VPNProcess struct {
	Port    int
	INChan  chan []byte      //from interface to udp
	OUTChan chan []byte      //from udp to interface
	Iface   *water.Interface //interface for our server
}

func NewServer(iface *water.Interface, port int) VPNProcess {
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
		n, err := s.Iface.Write(data)
		if err != nil {
			log.Println("Problem writing:", n, err)
		}
	}
}
func (s VPNProcess) ifaceToChan() {
	packet := make([]byte, BUFFERSIZE)
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
	retip := map[string]*net.UDPAddr{}
	go func() {
		buf := make([]byte, BUFFERSIZE)
		for {
			n, addr, err := conn.ReadFromUDP(buf)
			header, _ := ipv4.ParseHeader(buf[:n])
			fmt.Printf("[UDP -> OUTChan]\tReceived %d bytes from %v: %v %v %v %v\n", n, addr, header.Src, header.Dst, header.ID, header.Len)
			if err != nil || n == 0 {
				fmt.Println("Error: ", err)
				continue
			}
			retip["0"] = addr
			s.OUTChan <- buf[:n]
		}
	}()
	for data := range s.INChan {
		header, _ := ipv4.ParseHeader(data)
		if len(retip) > 0 {
			caddr := retip["0"]
			fmt.Printf("[INChan -> UDP]\t\tWriting %d bytes from %v: %v %v %v %v\n", len(data), caddr, header.Src, header.Dst, header.ID, header.Len)
			conn.WriteToUDP(data, caddr)
		}

	}
}

//ProcessClient process some client stuff
func (s *VPNProcess) ProcessClient(lip *net.UDPAddr, rip *net.UDPAddr) {
	conn, err := net.DialUDP("udp", lip, rip)
	if err != nil {
		log.Fatalln("Unable to get UDP socket:", err)
	}
	go func() {
		packet := make([]byte, BUFFERSIZE)
		for {
			plen, err := conn.Read(packet)
			if err != nil {
				break
			}
			header, _ := ipv4.ParseHeader(packet[:plen])
			fmt.Printf("[INChan -> UDP]\t\tReceiving %d bytes from %v: %v %v %v\n", plen, header.Src, header.Dst, header.ID, header.Len)
			s.OUTChan <- packet[:plen]
		}
	}()
	for data := range s.INChan {
		header, _ := ipv4.ParseHeader(data)
		fmt.Printf("[INChan -> UDP]\t\tWriting %d bytes to %v: %v %v %v %v\n", len(data), rip, header.Src, header.Dst, header.ID, header.Len)
		conn.WriteToUDP(data, rip)
	}
}
