package dns

import (
	//"github.com/nargesbyt/dnsresolver/dns/cache"
	"errors"
	"fmt"
	"net"
	"strings"
	"github.com/nargesbyt/dnsresolver/dns/cache"
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
		return header, nil, fmt.Errorf("unable to parse packet: %w", err)
	}

	//Question parses a single question
	question, err := p.Question()
	if err != nil {
		return header, &question, fmt.Errorf("unable to parse question part of packet: %w", err)
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
//this function get tld and return associated nameservers
//at first search it in cache if it doesn't exist it returns rootservers 
func (r *Resolver) GetNameServers(key string) ([]string, error) {
	if !r.Cache.Exists(key) {
		return RootServers, nil
	}

	value, err := r.Cache.Get(key)
	if err != nil {
		return nil, fmt.Errorf("unable to locate a cache record: %w", err)
	}
	return value.([]string), nil
}

func (r *Resolver) HandlePacket(packet Packet) error {
	header, question, err := packet.parsePacket()
	if err != nil {
		return err
	}

	//q holds a dns question
	q := r.SetQuestion(question)
	splited := strings.Split(q.Name, ".")
	key := splited[1]

	servers, err := r.GetNameServers(key)
	if err != nil {
		return err
	}

	response, err := r.resolver(servers, q)
	if err != nil {
		return fmt.Errorf("unable to resolve domain: %w", err)
	}

	response.MsgHdr.Id = header.ID
	err = packet.sendResponse(response)
	if err != nil {
		return err
	}

	return nil
}

func (r *Resolver) resolver(servers []string, question dns.Question) (*dns.Msg, error) {
	resp, err := r.dnsQuery(servers, question.Name, question.Qtype)
	switch {
	case err != nil:
		{
			return resp, fmt.Errorf("unable to resolve domain: %w", err)
		}

	//if the Answer section is not empty it contains the IP addresses of requested domain
	case len(resp.Answer) > 0:
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
			servers, err = r.lookupNameservers(resp)
			if err != nil {
				return nil, err
			}

			//splited := strings.Split(question.Name, ".")
			//key := splited[len(splited)-2]
			//r.Cache.Set(key, servers)
			//log.Info().Msg("data stored in cache")

			if len(servers) > 0 {
				resp, err := r.resolver(servers, question)
				if err != nil {
					return resp, fmt.Errorf("unable resolve domain: %w", err)
				}

				return resp, nil
			} else {
				resp.Rcode = dns.RcodeNameError
				return resp, nil
			}
		}
	}
}

func (r *Resolver) dnsQuery(servers []string, question string, qType uint16) (*dns.Msg, error) {
	message := new(dns.Msg)
	message.SetQuestion(dns.Fqdn(question), qType)

	c := new(dns.Client)

	for _, server := range servers {
		responseMessage, _, err := c.Exchange(message, server+":53")
		if err != nil {
			return nil, err
		}

		return responseMessage, nil
	}

	return nil, errors.New("no response from servers")
}

// lookupNameservers find the IP Address of nameservers
func (r *Resolver) lookupNameservers(message *dns.Msg) ([]string, error) {
	nameservers := make([]string, 0)

	// if Extra section is not empty it contains the IP Address of nameservers
	ns, extra := message.Ns, message.Extra

	for _, rr := range ns {
		/*if there is no record associated with requested type then in Ns section there is SOA record*/
		if rr.Header().Rrtype == dns.TypeSOA {
			return nil, nil
		}
		nameservers = append(nameservers, rr.(*dns.NS).Ns)
	}

	newServerFound := false
	var servers []string

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
		return servers, nil
	} else {
		for _, nameserver := range nameservers {
			if !newServerFound {
				//TODO checkCache
				resp, err := r.resolver(RootServers, dns.Question{Name: nameserver, Qtype: dns.TypeA, Qclass: dns.ClassINET})
				if err != nil {
					return nil, err
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

	return servers, nil
}
