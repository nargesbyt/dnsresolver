package main

import (
	dnsresolve "dnsResolver/dns"
	"fmt"
	"log"
	"net"
)

func main() {
	fmt.Println("starting a dns server ...")
	packetConnection, err := net.ListenPacket("udp", ":8080")
	if err != nil {
		log.Fatal(err)
		//panic(err)

	}
	defer packetConnection.Close()
	for {
		buf := make([]byte, 512)
		bytesRead, addr, err := packetConnection.ReadFrom(buf)
		if err != nil {
			//fmt.Printf("read error from%s:%s", addr.String(), err)
			log.Fatal(err)
			continue
		}
		 dnsresolve.HandlePacket(packetConnection, addr, buf[:bytesRead])

	}

}
