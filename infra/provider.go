package infra

import (
	"bufio"
	"bytes"
	"gossip/sipmsg"
	"hash/adler32"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
)

// Closer: need a means to shut down internet TCP connections
type Closer interface {
	Close() error
}

const (
	BufSize = 131072 // 128 KByte
)

var (
	// Maps need a lock if used concurrently
	networks  = make(map[string]Closer)
	mNetworks sync.Mutex
	// maps of channels that can transmit messages
	remoteEPs  = make(map[string]chan *sipmsg.Item)
	mRemoteEPs sync.Mutex
	//
	retransmit  = make(map[string]*sipmsg.Item)
	mRetransmit sync.Mutex
	//
	viaMsgs  = make(map[string]*sipmsg.Item)
	mViaMsgs sync.Mutex
	// indicator that interfaces are being closed - not an error
	closing bool
)

func addNetwork(provider string, cl Closer) {
	mNetworks.Lock()
	networks[provider] = cl
	mNetworks.Unlock()
}

func addRemoteEP(provider string, ch chan *sipmsg.Item) {
	mRemoteEPs.Lock()
	remoteEPs[provider] = ch
	mRemoteEPs.Unlock()
}

// ScanPost reads one post from the scanner and gives it to the director
func ScanPost(sc *bufio.Scanner, lAddr, rAddr net.Addr, ch chan *sipmsg.Item) (err error) {
	msg := new(sipmsg.SipMsg)
	msg.Direction = sipmsg.DirIn
	hash := adler32.New()
	// first line is already scanned
	msg.StartLine = sc.Text()
	hash.Write([]byte(msg.StartLine))
	for sc.Scan() {
		str := sc.Text()
		hash.Write([]byte(str))
		parts := strings.SplitN(str, ":", 2)
		if len(parts) == 2 {
			msg.Headers[parts[0]] = append(msg.Headers[parts[0]], parts[1])
		} else {
			// empty line without a header
			break
		}
	}
	// find out how much to read from the scanner
	cla := msg.Headers["Content-Length"]
	l := 0
	if len(cla) > 0 {
		cl := strings.TrimSpace(cla[0])
		l, err = strconv.Atoi(cl)
		if err != nil {
			return
		}
	}
	// start reading
	got := 0
	// only read the requested amount
	for got < l && sc.Scan() {
		str := sc.Text()
		hash.Write([]byte(str))
		got += len(str) + 2
		msg.BodyList = append(msg.BodyList, str)
	}
	// stopping retransmissions
	vh := msg.Headers["Via"]
	if vh != nil {
		m := viaReg.FindStringSubmatch(vh[0])
		if m != nil && len(m) > 1 {
			via := m[1]
			mViaMsgs.Lock()
			msg2 := viaMsgs[via]
			msg2.RetrCount = sipmsg.NoRetrans
			delete(viaMsgs, via)
			mViaMsgs.Unlock()
		}
	}
	// preparing for sending it to the director
	item := new(sipmsg.Item)
	item.Msg = msg
	item.Hash = hash.Sum32()
	item.Ch = ch
	item.LocalEP = lAddr.Network() + "/" + lAddr.String()
	item.RemoteEP = rAddr.Network() + "/" + rAddr.String()

	// the director will handle it and send it to the right analyser
	//  TODO DirectItem(item)
	return
}

// NewProvider creates provider structures and starts goroutine
func NewProvider(provider string) {
	// provider is like described below
	parts := strings.Split(provider, "/")
	if len(parts) != 2 {
		log.Fatal("provider string should be like 'udp/192.168.0.8:5060', got " + provider)
	}
	// udp or tcp
	switch parts[0] {
	// UDP
	case "udp":
		p, err := newUdpProvider(provider)
		if err != nil {
			log.Fatal(err)
		}
		go p.Sender()
		go p.Receiver()
		// TCP
	case "tcp":
		p, err := newTcpProvider(provider)
		if err != nil {
			log.Fatal(err)
		}
		go p.Sender()
		go p.Receiver()
	default:
		log.Fatal("couldn't handle network " + parts[0])
	}
}

// newUdpProvider creates a new UDP provider (only structure)
func newUdpProvider(provider string) (p *UdpProvider, err error) {
	p = new(UdpProvider)
	p.ch = make(chan *sipmsg.Item, 8)
	parts := strings.Split(provider, "/")
	nc, err := net.ListenPacket("udp", parts[1])
	p.netConn = nc
	addNetwork(provider, nc)
	addRemoteEP(parts[0], p.ch)
	return
}

// UdpProvider combines a network interface with a function that sends items from the channel to the net
type UdpProvider struct {
	ch      chan *sipmsg.Item // for items to send over this port
	netConn net.PacketConn
}

// Sender: receives items from the internal system and send it out on the channel, one after the other
func (p *UdpProvider) Sender() {
	for item := range p.ch {
		if item == nil {
			continue
		}
		log.Fatal("### not yet implemented")
		// TODO: implement sending it over UDP
	}
}

// Receiver for UDP, all messages come in the same way
func (p *UdpProvider) Receiver() {
	// local side is fixed
	lAddr := p.netConn.LocalAddr()
	buf := make([]byte, BufSize)
	for {
		// stops when channel is closed
		n, rAddr, err := p.netConn.ReadFrom(buf)
		if err != nil {
			if closing {
				return // ends goroutine
			}
			log.Fatal(err)
		}
		bb := buf[:n] // need a new slice
		scan := bufio.NewScanner(bytes.NewReader(bb))
		// send it to the general message reading routine
		scan.Scan() // scanning the first line, ScanPost starts with Text()
		err = ScanPost(scan, lAddr, rAddr, p.ch)
		if err != nil {
			log.Fatal(err)
		}
	}
	// the local end is closed
}

// newTcpProvider creates a new TCP provider (only structure)
func newTcpProvider(provider string) (p *TcpProvider, err error) {
	p = new(TcpProvider)
	p.ch = make(chan *sipmsg.Item, 8)
	p.conns = make(map[string]*net.TCPConn)
	parts := strings.Split(provider, "/")
	addr, err := net.ResolveTCPAddr("tcp", parts[1])
	if err != nil {
		log.Fatal(err)
	}
	// creating the listener
	nc, err := net.ListenTCP("tcp", addr)
	p.netConn = nc
	addNetwork(provider, nc)
	addRemoteEP(parts[0], p.ch)
	return
}

// TcpProvider combines a listener with multiple active connections
type TcpProvider struct {
	ch      chan *sipmsg.Item // for items to be send by this provider
	netConn *net.TCPListener  // listener for incoming connections
	//
	conns  map[string]*net.TCPConn // map of alive connections
	mConns sync.Mutex              // synchronizes access to conns
}

// Sender: receives items on the channel and sends it over the net
func (p *TcpProvider) Sender() {
	for item := range p.ch {
		if item == nil {
			continue
		}
		// TODO: implement sending on this provider
		// need to find out, if there is an existing connection - then using this one
		// otherwise, creating a new connection
	}
}

// Receiver accepts new connections on the listener
func (p *TcpProvider) Receiver() {
	for {
		conn, err := p.netConn.AcceptTCP()
		if err != nil {
			if closing {
				return // ends goroutinge
			}
			log.Fatal(err)
		}
		// start a new goroutine for receiving messages
		go p.ReceiveStream(conn)
	}
}

// ReceiveStream takes messages from connected streams
// streams are both from acceptance by the listener, or actively established
func (p *TcpProvider) ReceiveStream(conn *net.TCPConn) {
	laddr := conn.LocalAddr()
	raddr := conn.RemoteAddr()
	// TODO add remote address to a dictionary
	hp := raddr.String()
	p.mConns.Lock()
	p.conns[hp] = conn
	p.mConns.Unlock()
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		// will fail when stream is closed
		err := ScanPost(scanner, laddr, raddr, p.ch)
		if err != nil {
			log.Fatal(err)
		}
	}
	p.mConns.Lock()
	delete(p.conns, hp)
	p.mConns.Unlock()
}

// EndProviders: shuts down all connections
func CloseProviders() {
	log.Println("closing network interfaces")
	closing = true
	mNetworks.Lock()
	for _, c := range networks {
		c.Close()
	}
	mNetworks.Unlock()
}

func Transmit(item *sipmsg.Item) {
	mRemoteEPs.Lock()
	defer mRemoteEPs.Unlock()
	ch, ok := remoteEPs[item.RemoteEP]
	if !ok {
		// log.Println("could not find remote endpoint " + item.RemoteEP)
		parts := strings.Split(item.RemoteEP, "/")
		if len(parts) < 2 {
			log.Fatal("should have had transport/addr, got instead " + item.RemoteEP)
		}
		ch, ok = remoteEPs[parts[0]]
		if !ok {
			log.Fatal("could not find a remote endpoitn for " + parts[0])
		}
	}
	ch <- item
}
