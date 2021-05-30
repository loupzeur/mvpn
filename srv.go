package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"net"
	"sync"

	"github.com/lucas-clemente/quic-go"
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

func (s *VPNProcess) ProcessServerQuic() {
	listener, err := quic.ListenAddr(fmt.Sprintf(":%v", s.Port), generateTLSConfig(), nil)

	if nil != err {
		log.Fatalln("Unable to listen on UDP socket:", err)
	}

	for {
		sess, err := listener.Accept(context.Background())
		if err != nil {
			return
		}
		stream, err := sess.AcceptStream(context.Background())
		if err != nil {
			panic(err)
		}
		go func() {
			buf := make([]byte, *MTU)
			for {
				n, err := stream.Read(buf)
				if err != nil || n == 0 {
					fmt.Println("Error: ", err)
					continue
				}
				s.OUTChan <- buf[:n]
			}
		}()
		go func() {
			for data := range s.INChan {
				stream.Write(data)
			}
		}()
	}
}

//ProcessClient process some client stuff
func (s *VPNProcess) ProcessClientQuic(lip *net.UDPAddr, rip *net.UDPAddr, wg *sync.WaitGroup) {
	defer wg.Done()
	tlsConf := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"quic-echo-example"},
	}
	conn, err := net.ListenUDP("udp", lip)
	if err != nil {
		log.Fatalln("Unable to get UDP socket:", err)
	}
	log.Println("Listening on :", lip.String())
	session, err := quic.Dial(conn, rip, "", tlsConf, nil)
	if err != nil {
		return
	}
	stream, err := session.OpenStream()
	if err != nil {
		return
	}
	go func() {
		packet := make([]byte, *MTU)
		for {
			plen, err := stream.Read(packet)
			if err != nil {
				break
			}
			s.OUTChan <- packet[:plen]
		}
	}()
	for data := range s.INChan {
		n, err := stream.Write(data)
		if err != nil {
			log.Println("Error writing packet ", n, err)
		}
	}
}

// Setup a bare-bones TLS config for the server
func generateTLSConfig() *tls.Config {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		panic(err)
	}
	template := x509.Certificate{SerialNumber: big.NewInt(1)}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		panic(err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		panic(err)
	}
	return &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		NextProtos:   []string{"quic-echo-example"},
	}
}
