package core

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
)

type Protocol int

const (
	ProtocolTCP Protocol = iota + 1
	ProtocolUDP
	ProtocolICMP
)

type Service int

const (
	ServiceUnknown Service = iota + 1
	ServiceIRC
	ServiceDHCP
	ServiceDNS
	ServiceHTTP
	ServiceSSL
	ServiceSSH
)

type ConnState int

const (
	ConnStateS0 ConnState = iota + 1
	ConnStateS1
	ConnStateS2
	ConnStateS3
	ConnStateSF
	ConnStateRSTR
	ConnStateRSTO
	ConnStateSHR
	ConnStateOTH
	ConnStateREJ
	ConnStateSH
	ConnStateRSTOS0
	ConnStateRSTRH
)

type Attack int

const (
	AttackNone Attack = iota + 1
	AttackCnC
	AttackPartOfAHorizontalPortScan
	AttackDDoS
	AttackAttack
	AttackCnCFileDownload
)

func ParseService(v string) (Service, bool) {
	switch v {
	case "-":
		return ServiceUnknown, true
	case "irc":
		return ServiceIRC, true
	case "dhcp":
		return ServiceDHCP, true
	case "dns":
		return ServiceDNS, true
	case "http":
		return ServiceHTTP, true
	case "ssl":
		return ServiceSSL, true
	case "ssh":
		return ServiceSSH, true
	}

	return 0, false
}

func ParseProtocol(v string) (Protocol, bool) {
	switch v {
	case "tcp":
		return ProtocolTCP, true
	case "udp":
		return ProtocolUDP, true
	case "icmp":
		return ProtocolICMP, true
	}

	return 0, false
}

func ParseConnState(v string) (ConnState, bool) {
	switch v {
	case "S0":
		return ConnStateS0, true
	case "S1":
		return ConnStateS1, true
	case "S2":
		return ConnStateS2, true
	case "S3":
		return ConnStateS3, true
	case "SF":
		return ConnStateSF, true
	case "RSTR":
		return ConnStateRSTR, true
	case "RSTO":
		return ConnStateRSTO, true
	case "SHR":
		return ConnStateSHR, true
	case "OTH":
		return ConnStateOTH, true
	case "REJ":
		return ConnStateREJ, true
	case "SH":
		return ConnStateSH, true
	case "RSTOS0":
		return ConnStateRSTOS0, true
	case "RSTRH":
		return ConnStateRSTRH, true
	}

	return 0, false
}

func ParseAttack(v string) (Attack, bool) {
	switch v {
	case "-":
		return AttackNone, true
		//	case "C&C":
		//		return AttackCnC, true
	case "PartOfAHorizontalPortScan":
		return AttackPartOfAHorizontalPortScan, true
		//	case "DDoS":
		//		return AttackDDoS, true
		//	case "Attack":
		//		return AttackAttack, true
		//	case "C&C-FileDownload":
		//		return AttackCnCFileDownload, true
	}

	return 0, false
}

func (a Attack) String() string {
	switch a {
	case AttackCnC:
		return "C&"
	case AttackPartOfAHorizontalPortScan:
		return "HorizontalPortScan"
	case AttackDDoS:
		return "DDoS"
	default:
		return "None"
	}
}

type Meta struct {
	IP      string
	Port    int64
	Bytes   int64
	Packets int64
	IPBytes int64
}

type Entry struct {
	UUID      string
	Timestamp string

	Origin      Meta
	Response    Meta
	ProtoCount  int
	Protocol    Protocol
	Service     Service
	Duration    string
	ConnState   ConnState
	MissedBytes int64
	Malicious   bool
	Attack      Attack
}

func NewFromLines(lines ...string) []Entry {
	var (
		entries           []Entry
		visitedProtocols  = make(map[string]struct{})
		visitedServices   = make(map[string]struct{})
		visitedConnStates = make(map[string]struct{})
		visitedAttacks    = make(map[string]struct{})
		visitedStatuses   = make(map[string]struct{})
	)

	for _, line := range lines {
		line = strings.TrimSuffix(line, "\r")

		// Remove \r at the end of the line.
		parts := strings.Split(line, "\t")
		if len(parts) != 21 {
			continue
		}

		// Last three columns are separated by spaces.
		last := parts[20]
		parts = parts[:20]
		parts = append(parts, strings.Split(last, " ")...)

		// Add minus as a default value.
		for i, part := range parts {
			if part == "" {
				parts[i] = "-"
			}
		}

		protocol, ok := ParseProtocol(parts[6])
		if !ok {
			if parts[6] != "-" {
				_, ok := visitedProtocols[parts[6]]
				if !ok {
					logrus.Info("new protocol type ", parts[6])
					visitedProtocols[parts[6]] = struct{}{}
				}
			}

			continue
		}

		service, ok := ParseService(parts[7])
		if !ok {
			_, ok := visitedServices[parts[7]]
			if !ok {
				logrus.Info("new service type ", parts[7])
				visitedServices[parts[7]] = struct{}{}
			}

			continue
		}

		originBytes, err := strconv.ParseInt(parts[9], 10, 64)
		if err != nil {
			continue
		}

		responseBytes, err := strconv.ParseInt(parts[10], 10, 64)
		if err != nil {
			continue
		}

		connState, ok := ParseConnState(parts[11])
		if !ok {
			if parts[11] != "-" {
				_, ok := visitedConnStates[parts[11]]
				if !ok {
					logrus.Info("new state type ", parts[11])
					visitedConnStates[parts[11]] = struct{}{}
				}
			}

			continue
		}

		missedBytes, err := strconv.ParseInt(parts[14], 10, 64)
		if err != nil {
			continue
		}

		originPackets, err := strconv.ParseInt(parts[16], 10, 64)
		if err != nil {
			continue
		}

		originIPBytes, err := strconv.ParseInt(parts[17], 10, 64)
		if err != nil {
			continue
		}

		responsePackets, err := strconv.ParseInt(parts[18], 10, 64)
		if err != nil {
			continue
		}

		responseIPBytes, err := strconv.ParseInt(parts[19], 10, 64)
		if err != nil {
			continue
		}

		parts[23] = strings.ToLower(parts[23])
		if parts[23] != "malicious" && parts[23] != "benign" {
			if parts[23] != "-" {
				_, ok := visitedStatuses[parts[23]]
				if !ok {
					logrus.Info("new status type ", parts[23])
					visitedStatuses[parts[23]] = struct{}{}
				}
			}

			continue
		}

		attack, ok := ParseAttack(parts[26])
		if !ok {
			if parts[26] != "-" {
				_, ok := visitedAttacks[parts[26]]
				if !ok {
					logrus.Info("new attack type ", parts[26])
					visitedAttacks[parts[26]] = struct{}{}
				}
			}

			continue
		}

		originPort, err := strconv.ParseInt(parts[3], 10, 64)
		if err != nil {
			continue
		}

		responsePort, err := strconv.ParseInt(parts[5], 10, 64)
		if err != nil {
			continue
		}

		entries = append(entries, Entry{
			UUID:      parts[1],
			Timestamp: parts[0],
			Origin: Meta{
				IP:      parts[2],
				Port:    originPort,
				Bytes:   originBytes,
				Packets: originPackets,
				IPBytes: originIPBytes,
			},
			Response: Meta{
				IP:      parts[4],
				Port:    responsePort,
				Bytes:   responseBytes,
				Packets: responsePackets,
				IPBytes: responseIPBytes,
			},
			Protocol:    protocol,
			Service:     service,
			Duration:    parts[8],
			ConnState:   connState,
			MissedBytes: missedBytes,
			Malicious:   parts[23] == "malicious",
			Attack:      attack,
		})
	}

	return entries
}

func (e Entry) Floats() map[int]float64 {
	return map[int]float64{
		1:  float64(e.Origin.IPBytes),
		2:  float64(e.Origin.Packets),
		3:  float64(e.Origin.Bytes),
		4:  float64(e.Response.Port),
		5:  float64(e.Response.IPBytes),
		6:  float64(e.Response.Packets),
		7:  float64(e.Response.Bytes),
		8:  float64(e.ProtoCount),
		9:  float64(e.Protocol),
		10: float64(e.Service),
		11: float64(e.ConnState),
		12: float64(e.MissedBytes),
	}
}

func (e Entry) String() string {
	malicious := "-"
	if e.Malicious {
		malicious = "+"
	}

	return fmt.Sprintf(
		"%s1 1:%d 2:%d 3:%d 4:%d 5:%d 6:%d 7:%d 8:%d 9:%d 10:%d 11:%d 12:%d",
		malicious,
		e.Origin.IPBytes,
		e.Origin.Packets,
		e.Origin.Bytes,
		e.Response.Port,
		e.Response.IPBytes,
		e.Response.Packets,
		e.Response.Bytes,
		e.ProtoCount,
		e.Protocol,
		e.Service,
		e.ConnState,
		e.MissedBytes,
	)
}
