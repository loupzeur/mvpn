package vpn

import (
	"crypto/tls"
	"log"
	"net"
	"strings"
	"sync"

	"github.com/loupzeur/mvpn/utils"
	"github.com/lucas-clemente/quic-go"
)

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
	//todo use remote gateway address
	go utils.ContinuousPing("10.9.0.1")
	go func() {
		packet := make([]byte, MTU)
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
