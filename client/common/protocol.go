package common

import (
	"fmt"
	"net"
)

// Protocol Encapsulates how to send and receive data throw a socket
type Protocol struct {
	conn net.Conn
}

// NewProtocol Initializes a new protocol receiving the socket as a parameter
func NewProtocol(conn net.Conn) *Protocol {
	protocol := &Protocol{
		conn: conn,
	}
	return protocol
}

// CloseWith Double Dispatch | Close this protocol's underlying socket using the given closer function
func (p *Protocol) CloseWith(closureToClose func(net.Conn)) {
	closureToClose(p.conn)
}

// SendString Sends a string through the socket
// First, sends 2 bytes (big endian) that indicate the length, then sends the payload of that length
// Returns an error if the string is too long or fails
func (p *Protocol) SendString(s string) error {
	payloadSize := len(s)
	if payloadSize > 0xFFFF {
		return fmt.Errorf(
			"string too long for protocol, max is 65535, current: %d, string: %s",
			payloadSize,
			s,
		)
	}
	bigEndianSize := uint16(payloadSize)

	// 2 bytes + el largo del payload
	buf := make([]byte, 2+payloadSize)

	// Guardo 2 bytes sin signo en formato big endian indicando el largo del string.
	buf[0] = byte(bigEndianSize >> 8)
	buf[1] = byte(bigEndianSize)

	// Copio el contenido del string 2 bytes mas adelante
	copy(buf[2:], s)

	return p.sendAll(buf)
}

// WaitUint8 Wait a single unsigned byte (0-255) through the socket
// Returns the byte value or an error if fails
func (p *Protocol) WaitUint8() (uint8, error) {
	var buf [1]byte

	read, err := p.conn.Read(buf[0:])
	if err != nil {
		return 0, err
	}
	if read == 0 {
		return 0, fmt.Errorf("BrokenPipeError")
	}

	return buf[0], nil
}

// sendAll Send the entire contents of buf through the socket
// Retries until the entire buffer is sent
// Return error if the connection is closed unexpectedly
func (p *Protocol) sendAll(buf []byte) error {
	sz := 0
	for sz < len(buf) {
		written, err := p.conn.Write(buf[sz:])
		if err != nil {
			return err
		}
		sz += written
	}
	return nil
}
