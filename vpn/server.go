package vpn

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/loupzeur/mvpn/utils"
	"github.com/lucas-clemente/quic-go"
	//https://github.com/buger/goterm
)

//ProcessServerQuic process QUIC related packets on server sides
func (s *VPNProcess) ProcessServerQuic() {
	listener, err := quic.ListenAddr(fmt.Sprintf("0.0.0.0:%v", s.Port), utils.GenerateTLSConfig(), nil)

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
			buf := make([]byte, MTU)
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
