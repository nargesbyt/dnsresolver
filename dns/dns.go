package dnsresolve

import (
	"errors"
	"fmt"
	"log"
	"net"
	"strings"

	"github.com/miekg/dns"
	"golang.org/x/net/dns/dnsmessage"
)

const ROOT_SERVERS = "198.41.0.4,199.9.14.201,192.33.4.12,199.7.91.13,192.203.230.10,192.5.5.241,192.112.36.4,198.97.190.53,192.36.148.17,192.58.128.30,193.0.14.129,199.7.83.42,202.12.27.33"

func getRootServers() []string {
	rootServers := []string{}
	for _, rootServer := range strings.Split(ROOT_SERVERS, ",") {
		rootServers = append(rootServers, rootServer)
	}
	return rootServers
}

func HandlePacket(pc net.PacketConn, addr net.Addr, buf []byte) error {
	var p dnsmessage.Parser
	header, err := p.Start(buf)
	if err != nil {
		log.Fatal(err)
		//fmt.Println("unable to parse packet")
		return err
	}

	question, err := p.Question()

	if err != nil {
		log.Fatal(err)
		return err
	}
	var q dns.Question

	q.Name = question.Name.String()
	q.Qtype = uint16(question.Type)
	q.Qclass = uint16(question.Class)

	response, err := resolver(getRootServers(), q)
	if err != nil {
		log.Fatal(err)
		return err
	}

	response.MsgHdr.Id = header.ID
	responseBuffer, err := response.Pack()
	if err != nil {
		return err
	}
	_, err = pc.WriteTo(responseBuffer, addr)
	if err != nil {
		log.Println("unable to write response message on connection")
		return err
	}

	return nil

}

func resolver(servers []string, question dns.Question) (*dns.Msg, error) {

	resp, err := dnsQuery(servers, question.Name, question.Qtype)
	switch {
	case err != nil:
		{
			log.Println(err)
			return resp, err
		}

	case len(resp.Answer) > 0:
		return resp, nil

	case len(resp.Ns) == 0:
		{
			resp.Rcode = dns.RcodeNameError
			return resp, nil
		}
	default:
		{
			servers := lookupNameservers(resp)
			if len(servers) > 0 {
				resp, err := resolver(servers, question)
				if err != nil {
					return resp, err
				}
				return resp, nil
			} else {
				resp.Rcode = dns.RcodeNameError
				return resp, nil
			}
		}

	}
	return resp, nil

}

func dnsQuery(servers []string, question string, qType uint16) (*dns.Msg, error) {

	message := new(dns.Msg)
	message.SetQuestion(dns.Fqdn(question), qType)

	c := new(dns.Client)

	for _, server := range servers {
		responseMessage, _, err := c.Exchange(message, server+":53")
		if err == nil {
			return responseMessage, nil
		}
	}
	return nil, errors.New("no response from servers")
}

func lookupNameservers(message *dns.Msg) (servers []string) {
	nameservers := []string{}
	ns, extra := message.Ns, message.Extra

	for _, rr := range ns {
		nameservers = append(nameservers, rr.(*dns.NS).Ns)
	}

	newServerFound := false
	servers = []string{}

	for _, rr := range extra {
		if rr.Header().Rrtype == dns.TypeA {
			for _, nameserver := range nameservers {
				if rr.Header().Name == nameserver {
					newServerFound = true
					servers = append(servers, (rr.(*dns.A).A[:]).String())
				}

			}
		}
	}
	if newServerFound {
		return servers

	} else {
		for _, nameserver := range nameservers {
			if !newServerFound {
				resp, err := resolver(getRootServers(), dns.Question{Name: nameserver, Qtype: dns.TypeA, Qclass: dns.ClassINET})
				if err != nil {
					fmt.Printf("warning: lookup of nameserver %s failed%s \n", nameserver, err)
				} else {
					newServerFound = true
					for _, answer := range resp.Answer {
						if answer.Header().Rrtype == dns.TypeA {
							servers = append(servers, answer.(*dns.A).A[:].String())
						}
					}
				}
			}
		}
	}
	return servers
}
