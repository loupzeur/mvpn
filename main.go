package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"

	"github.com/songgao/water"
)

const (
	// I use TUN interface, so only plain IP packet, no ethernet header + mtu is set to 1300
	BUFFERSIZE = 1500
	MTU        = "1300"
)

var (
	typeVPN  = flag.String("type", "server", "server or client depending on configuration")
	localIP  = flag.String("local", "", "Local tun interface IP/MASK like 192.168.0.1⁄24")
	remoteIP = flag.String("remote", "", "Remote server (external) IP like 8.8.8.8")
	localIPs = flag.String("localips", "", "List of IP of interface to use for aggregation")
	port     = flag.Int("port", 43210, "UDP port for communication")
)

func runIP(args ...string) {
	cmd := exec.Command("/sbin/ip", args...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	err := cmd.Run()
	if nil != err {
		log.Fatalln("Error running /sbin/ip:", err)
	}
}

func main() {
	flag.Parse()
	if localIP == nil || *localIP == "" {
		flag.Usage()
		log.Fatalln("\nlocal ip is not specified")
	}
	iface, err := water.New(water.Config{DeviceType: water.TUN})
	if nil != err {
		log.Fatalln("Unable to allocate TUN interface:", err)
	}
	log.Println("Interface allocated:", iface.Name())
	runIP("link", "set", "dev", iface.Name(), "mtu", MTU)
	runIP("addr", "add", *localIP, "dev", iface.Name())
	runIP("link", "set", "dev", iface.Name(), "up")

	srv := NewServer(iface, *port)
	srv.Run()
	if *typeVPN == "server" {
		srv.ProcessConnection()
	} else { //client stuff
		if remoteIP == nil || *remoteIP == "" {
			flag.Usage()
			log.Fatalln("\nremote server is not specified")
		}
		remote, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%v", *remoteIP, *port))
		if err != nil {
			log.Fatalln("Remote addr is not valid:", err)
		}
		if localIPs == nil || *localIPs == "" {
			flag.Usage()
			log.Fatalln("\nlocal ips are not specified")
		}
		for _, lip := range strings.Split(*localIPs, " ") {
			lipA, err := net.ResolveUDPAddr("udp", lip+":50000")
			if err != nil {
				log.Fatalln("Local addr", lip, "is not valid:", err)
			}
			srv.ProcessClient(lipA, remote)
		}
	}

}
