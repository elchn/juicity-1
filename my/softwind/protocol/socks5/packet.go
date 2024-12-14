// Modified from https://github.com/nadoo/glider/tree/v0.16.2

package socks5

import (
	"errors"
	"fmt"
	"net"
	"net/netip"

	"github.com/daeuniverse/softwind/netproxy"

	"github.com/daeuniverse/softwind/pool"
	"github.com/daeuniverse/softwind/protocol/infra/socks"
)

// PktConn .
type PktConn struct {
	netproxy.PacketConn
	ctrlConn  netproxy.Conn // tcp control conn
	target    string
	proxyAddr string
}

// NewPktConn returns a PktConn, the writeAddr must be *net.UDPAddr or *net.UnixAddr.
func NewPktConn(c netproxy.PacketConn, proxyAddr string, targetAddr string, ctrlConn netproxy.Conn) *PktConn {
	pc := &PktConn{
		PacketConn: c,
		target:     targetAddr,
		proxyAddr:  proxyAddr,
		ctrlConn:   ctrlConn,
	}

	if ctrlConn != nil {
		go func() {
			buf := pool.Get(1)
			defer pool.Put(buf)
			for {
				_, err := ctrlConn.Read(buf)
				if err, ok := err.(net.Error); ok && err.Timeout() {
					continue
				}
				// log.F("[socks5] dialudp udp associate end")
				return
			}
		}()
	}

	return pc
}

// ReadFrom overrides the original function from transport.PacketConn.
func (pc *PktConn) ReadFrom(b []byte) (int, netip.AddrPort, error) {
	n, _, target, err := pc.readFrom(b)
	return n, target, err
}

func (pc *PktConn) readFrom(b []byte) (int, netip.AddrPort, netip.AddrPort, error) {
	buf := pool.Get(len(b))
	defer pool.Put(buf)

	n, raddr, err := pc.PacketConn.ReadFrom(buf)
	if err != nil {
		return n, raddr, netip.AddrPort{}, err
	}

	if n < 3 {
		return n, raddr, netip.AddrPort{}, errors.New("not enough size to get addr")
	}

	// https://www.rfc-editor.org/rfc/rfc1928#section-7
	// +----+------+------+----------+----------+----------+
	// |RSV | FRAG | ATYP | DST.ADDR | DST.PORT |   DATA   |
	// +----+------+------+----------+----------+----------+
	// | 2  |  1   |  1   | Variable |    2     | Variable |
	// +----+------+------+----------+----------+----------+
	tgtAddr := socks.SplitAddr(buf[3:n])
	if tgtAddr == nil {
		return n, raddr, netip.AddrPort{}, errors.New("can not get target addr")
	}

	target, err := net.ResolveUDPAddr("udp", tgtAddr.String())
	if err != nil {
		return n, raddr, netip.AddrPort{}, errors.New("wrong target addr")
	}

	n = copy(b, buf[3+len(tgtAddr):n])
	return n, raddr, target.AddrPort(), err
}

// WriteTo overrides the original function from transport.PacketConn.
func (pc *PktConn) WriteTo(b []byte, addr string) (int, error) {
	target, err := socks.ParseAddr(addr)

	if err != nil {
		return 0, fmt.Errorf("invalid addr: %w", err)
	}

	tgtLen := len(target)
	buf := pool.Get(3 + tgtLen + len(b))
	defer pool.Put(buf)

	copy(buf, []byte{0, 0, 0})
	copy(buf[3:], target)
	copy(buf[3+tgtLen:], b)

	n, err := pc.PacketConn.WriteTo(buf, pc.proxyAddr)
	if n > tgtLen+3 {
		return n - tgtLen - 3, err
	}

	return 0, err
}

// Close .
func (pc *PktConn) Close() error {
	if pc.ctrlConn != nil {
		pc.ctrlConn.Close()
	}

	return pc.PacketConn.Close()
}

func (c *PktConn) Read(b []byte) (n int, err error) {
	n, _, err = c.ReadFrom(b)
	return
}

func (c *PktConn) Write(b []byte) (n int, err error) {
	return c.WriteTo(b, c.target)
}
