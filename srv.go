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
	"golang.org/x/net/ipv4"
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
	listener, err := quic.ListenAddr(fmt.Sprintf("0.0.0.0:%v", s.Port), generateTLSConfig(), nil)

	if nil != err {
		log.Fatalln("Unable to listen on UDP socket:", err)
	}

	for {
		sess, err := listener.Accept(context.Background())
		if err != nil {
			log.Fatalln("Unable to accept session:", err)
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
					if strings.Contains(err.Error(), "imeout") {
						//connection is closed client side
						stream.Close()
						break
					}
					fmt.Println("Error Read stream Server : ", err)
					continue
				}
				s.OUTChan <- buf[:n]
			}
		}()
		go func() {
			for data := range s.INChan {
				_, err := stream.Write(data)
				if err != nil && (strings.Contains(err.Error(), "imeout") ||
					strings.Contains(err.Error(), "losed")) {
					log.Println("Closing stream with : ", sess.RemoteAddr().String())
					break //stop this loop
				}
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
