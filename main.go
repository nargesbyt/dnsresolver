package main

import (
	"dnsResolver/dns"
	"dnsResolver/dns/cache"
	"errors"
	"fmt"
	"net"
	"os"

	//"git.mills.io/prologic/bitcask"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog/pkgerrors"
)

func checkCacheFolder() {
	if _, err := os.Stat(".cache"); errors.Is(err, os.ErrNotExist) {
		err := os.Mkdir(".cache", os.ModePerm)
		if err != nil {
			log.Err(err).Msg("unable to make directory")
		}
	}
}

func main() {
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
	log.Logger = zerolog.New(os.Stderr).
		Level(zerolog.InfoLevel).
		With().
		Timestamp().
		Caller().
		Logger()

	fmt.Println("starting a dns server ...")

	packetConnection, err := net.ListenPacket("udp", ":8080")
	if err != nil {
		log.Fatal().Err(err).Msg("unable to make connection")

	}
	defer packetConnection.Close()

	r := dns.Resolver{
		Cache: &cache.InMemoryCache{make(map[string]interface{})},
	}

	for {
		buf := make([]byte, 512)
		//ReadFrom read a packet from connection and copy the payload into buf
		bytesRead, addr, err := packetConnection.ReadFrom(buf)
		if err != nil {
			//fmt.Printf("read error from%s:%s", addr.String(), err)
			log.Fatal().Err(err).Msg("unable to read packet")
			continue
		}
		packet := dns.Packet{
			Conn: packetConnection,
			Address: addr,
			Body: buf[:bytesRead],
		}
		r.HandlePacket(packet)

	}

}
