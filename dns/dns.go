package dns

import (
	//"dnsResolver/dns/cache"
	"dnsResolver/dns/cache"
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/miekg/dns"
	"github.com/rs/zerolog/log"
	"golang.org/x/net/dns/dnsmessage"
)

var RootServers = []string{"198.41.0.4", "199.9.14.201", "192.33.4.12", "199.7.91.13", "192.203.230.10", "192.5.5.241", "192.112.36.4", "198.97.190.53", "192.36.148.17", "192.58.128.30", "193.0.14.129", "199.7.83.42", "202.12.27.33"}

type Resolver struct {
	Cache cache.Cache
}
type Packet struct {
	Conn    net.PacketConn
	Address net.Addr
	Body    []byte
}

func (pac *Packet) parsePacket() (dnsmessage.Header, *dnsmessage.Question, error) {
	var p dnsmessage.Parser
	//Start parses the header
	header, err := p.Start(pac.Body)

	if err != nil {
		log.Fatal().Err(err).Msg("unable to parse packet")
		return header, nil, err
	}
	//Question parses a single question
	question, err := p.Question()

	if err != nil {
		log.Fatal().Err(err).Msg("unable to parse question part of packet")
		return header, &question, err
	}
	return header, &question, nil
}

func (pac *Packet) sendResponse(response *dns.Msg) error {
	responseBuffer, err := response.Pack()
	if err != nil {
		return err
	}
	//writing response packet on the connection
	_, err = pac.Conn.WriteTo(responseBuffer, pac.Address)
	if err != nil {
		log.Err(err).Msg("unable to write response message on connection")
		return err
	}

	return nil
}
func (r *Resolver) SetQuestion(question *dnsmessage.Question) dns.Question {
	var q dns.Question
	q.Name = question.Name.String()
	q.Qtype = uint16(question.Type)
	q.Qclass = uint16(question.Class)
	return q
}

func (r *Resolver) CheckCache(key string) ([]string, error) {

	value, err := r.Cache.Get(key)
	if !errors.Is(err, cache.ErrKeyNotExists) {
		log.Err(err).Msg("unale to connect to cache")
		return nil, err
	}
	if err == nil {
		if stringValue, ok := value.([]string); ok {
			return stringValue, nil
		}
		return nil, nil
	}
	return nil, nil

}

func (r *Resolver) HandlePacket(packet Packet) error {

	header, question, _ := packet.parsePacket()
	//q holds a dns question
	q := r.SetQuestion(question)
	splited := strings.Split(q.Name, ".")
	key := splited[1]
	servers, err := r.CheckCache(key)
	var nameServers []string
	if err == nil && servers != nil {
		fmt.Printf("values associated to key are: %s", servers)
		nameServers = servers
	} else {
		log.Err(err).Msg("unable to read data from cache")
		nameServers = RootServers
	}
	response, err := r.resolver(nameServers, q)
	if err != nil {
		log.Fatal().Err(err).Msg("unable to resolve domain")
		return err
	}
	response.MsgHdr.Id = header.ID
	packet.sendResponse(response)
	return nil
}

func (r *Resolver) resolver(servers []string, question dns.Question) (*dns.Msg, error) {

	resp, err := r.dnsQuery(servers, question.Name, question.Qtype)
	switch {
	case err != nil:
		{
			log.Err(err).Msg("unable to resolve domain")
			return resp, err
		}
	//if the Answer section is not empty it contains the IP addresses of requested domain
	case len(resp.Answer) > 0:
		//r.Cache.Set(question.Name, resp.Answer)
		return resp, nil

	//if the Ns section is empty it means there is no nameserver to know the address of requested domain
	case len(resp.Ns) == 0:
		{
			resp.Rcode = dns.RcodeNameError
			return resp, nil
		}

	//if (len(Answer)==0 and len(Ns)>0) then there are some nameservers that we ask them where domain is
	default:
		{
			servers := r.lookupNameservers(resp)
			splited := strings.Split(question.Name, ".")
			key := splited[len(splited)-1]
			r.Cache.Set(key, servers)
			log.Info().Msg("data stored in cache")

			if len(servers) > 0 {
				resp, err := r.resolver(servers, question)
				if err != nil {
					log.Err(err).Msg("unable resolve domain")
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

func (r *Resolver) dnsQuery(servers []string, question string, qType uint16) (*dns.Msg, error) {
	log.Print("dnsQuery() ", servers)
	fmt.Printf("question: %s\n ", question)
	message := new(dns.Msg)
	message.SetQuestion(dns.Fqdn(question), qType)

	c := new(dns.Client)

	for _, server := range servers {
		responseMessage, _, err := c.Exchange(message, server+":53")
		fmt.Print("response: ", responseMessage)
		if err == nil {
			log.Print("response :", responseMessage)
			return responseMessage, nil

		}
		log.Print("exchange error : ", err.Error())
	}

	return nil, errors.New("no response from servers")
}

// lookupNameservers find the IP Address of nameservers
func (r *Resolver) lookupNameservers(message *dns.Msg) (servers []string) {
	nameservers := []string{}
	headerNameParts := strings.Split(message.Ns[0].Header().Name, ".")

	fmt.Printf("message : %#v \n", *message)
	fmt.Printf("ns : %#v \n", len(headerNameParts))
	fmt.Printf("qtype : %#v \n", message.Question[0].Qtype)
	fmt.Printf("qclass : %#v \n", message.Question[0].Qclass)
	// if Extra section is not empty it contains the IP Address of nameservers
	ns, extra := message.Ns, message.Extra

	for _, rr := range ns {
		/*if there is no record associated with requested type then in Ns section there is SOA record*/
		if rr.Header().Rrtype == dns.TypeSOA {
			return nil
		}
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
				//TODO checkCache
				resp, err := r.resolver(RootServers, dns.Question{Name: nameserver, Qtype: dns.TypeA, Qclass: dns.ClassINET})
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
