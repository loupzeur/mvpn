package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"

	"github.com/loupzeur/mvpn/utils"
	"github.com/loupzeur/mvpn/vpn"
	"github.com/songgao/water"
)

var (
	typeVPN  = flag.String("type", "server", "server or client depending on configuration")
	localIP  = flag.String("local", "", "Local tun interface IP/MASK like 192.168.0.1⁄24")
	remoteIP = flag.String("remote", "", "Remote server (external) IP like 8.8.8.8")
	localIPs = flag.String("localips", "", "List of IP of interface to use for aggregation")
	port     = flag.Int("port", 43210, "UDP port for communication")
	MTU      = flag.Int("mtu", 1440, "MTU for interface")
)

func main() {
	flag.Parse()
	if localIP == nil || *localIP == "" {
		flag.Usage()
		log.Fatalln("\nlocal ip is not specified")
	}

	//!todo to be removed
	//create a tunnel externaly to avoid root requirements
	iface, err := water.New(water.Config{DeviceType: water.TUN})
	if nil != err {
		log.Fatalln("Unable to allocate TUN interface:", err)
	}
	log.Println("Interface allocated:", iface.Name())
	utils.RunIP("link", "set", "dev", iface.Name(), "mtu", fmt.Sprintf("%d", *MTU))
	utils.RunIP("addr", "add", *localIP, "dev", iface.Name())
	utils.RunIP("link", "set", "dev", iface.Name(), "up")

	//end to be removed

	//connect the interface and channels
	vpn.MTU = *MTU //set MTU
	ifc := vpn.NewVPN(iface, *port)
	ifc.Run()
	if *typeVPN == "server" {
		ifc.ProcessServerQuic()
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
		wg := sync.WaitGroup{}
		for _, lip := range strings.Split(*localIPs, " ") {
			lipA, err := net.ResolveUDPAddr("udp", lip+":0")
			if err != nil {
				log.Fatalln("Local addr", lip, "is not valid:", err)
			}
			wg.Add(1)
			go ifc.ProcessClientQuic(lipA, remote, &wg)
		}
		wg.Wait()
	}
}
