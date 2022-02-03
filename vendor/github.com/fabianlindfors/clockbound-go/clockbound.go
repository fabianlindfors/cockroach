package clockbound

import (
	"encoding/binary"
	"net"
	"os"
	"syscall"
	"time"
)

const defaultSocket = "/run/clockboundd/clockboundd.sock"
const tempSocket = "/tmp/clockbound-go"

type Clock struct {
	socket net.Conn
}

func New() (*Clock, error) {
	return NewWithPath(defaultSocket)
}

func NewWithPath(path string) (*Clock, error) {
	if err := os.RemoveAll(tempSocket); err != nil {
		return nil, err
	}

	// Clockbound requires that we have our own local socket.
	// We create one by calling listen and then immediately closing.
	syscall.Umask(777)
	l, err := net.Listen("unixgram", tempSocket)
	if err != nil {
		return nil, err
	}
	l.Close()

	// Connect to the ClockboundD socket through our own socket
	var dialer net.Dialer
	dialer.LocalAddr = &net.UnixAddr{
		Name: tempSocket,
		Net:  "unixgram",
	}

	socket, err := dialer.Dial("unixgram", path)
	if err != nil {
		return nil, err
	}

	return &Clock{
		socket: socket,
	}, nil
}

func (c *Clock) Now() (Bounds, error) {
	// Request header:
	// 	  1: Version
	//	  2: Command type (1 = now)
	// 	3-4: Reserved
	request := [4]byte{1, 1, 0, 0}

	if _, err := c.socket.Write(request[:]); err != nil {
		return Bounds{}, err
	}

	c.socket.SetReadDeadline(time.Now().Add(time.Second * 10))

	var response [20]byte
	if _, err := c.socket.Read(response[:]); err != nil {
		return Bounds{}, err
	}

	earliest := binary.BigEndian.Uint64(response[4:12])
	latest := binary.BigEndian.Uint64(response[12:20])

	return Bounds{
		Earliest: earliest,
		Latest:   latest,
	}, nil
}

func (c *Clock) Before(beforeTime uint64) (bool, error) {
	// Request header:
	// 	  1: Version
	//	  2: Command type (2 = before)
	// 	3-4: Reserved
	request := [12]byte{1, 2}

	// 4-12: 8-byte big endian timestamp to compare
	binary.BigEndian.PutUint64(request[4:12], beforeTime)

	if _, err := c.socket.Write(request[:]); err != nil {
		return false, err
	}

	c.socket.SetReadDeadline(time.Now().Add(time.Second * 10))

	var response [20]byte
	if _, err := c.socket.Read(response[:]); err != nil {
		return false, err
	}

	before := response[4] != 0

	return before, nil
}

func (c *Clock) After(beforeTime uint64) (bool, error) {
	// Request header:
	// 	  1: Version
	//	  2: Command type (3 = after)
	// 	3-4: Reserved
	request := [12]byte{1, 3}

	// 4-12: 8-byte big endian timestamp to compare
	binary.BigEndian.PutUint64(request[4:12], beforeTime)

	if _, err := c.socket.Write(request[:]); err != nil {
		return false, err
	}

	c.socket.SetReadDeadline(time.Now().Add(time.Second * 10))

	var response [20]byte
	if _, err := c.socket.Read(response[:]); err != nil {
		return false, err
	}

	after := response[4] != 0
	return after, nil
}

type Bounds struct {
	Earliest uint64
	Latest   uint64
}
