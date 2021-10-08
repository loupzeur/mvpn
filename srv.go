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
	"strings"
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

//func that takes array of byte of ip header and return sequence number
func getSequence(data []byte) int {
	startTcp := 20
	for i := range data[20:60] {
		if i == 0x0F {
			startTcp += i
			break
		}
	}
	return int(data[startTcp+4])<<24 | int(data[startTcp+5])<<16 | int(data[startTcp+6])<<8 | int(data[startTcp+7])
}

func (s VPNProcess) chanToIface() {
	for data := range s.OUTChan {
		//check tcp
		if len(data) > 12 && data[12] == 0x06 {
			log.Printf("IP packet is TCP!!\n%d\n%+v\n", getSequence(data), data[:60])
			//will require reordering of the packet
		}
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

//ProcessServerQuic process QUIC related packets on server sides
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
starStream:
	conn, err := net.ListenUDP("udp", lip)
	if err != nil {
		log.Fatalln("Unable to get UDP socket:", err)
	}
	log.Println("Listening on :", lip.String())
	session, err := quic.Dial(conn, rip, "", tlsConf, nil)
	if err != nil {
		log.Fatalln("Unable to connect to server: ", rip.IP, rip.Port, err)
	}
	stream, err := session.OpenStream()
	if err != nil {
		log.Fatalln("Unable to open stream :", err)
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
			if strings.Contains(err.Error(), "imeout") {
				log.Println("Disconnected from timeout")
				goto starStream //reconnect to server
			}
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
