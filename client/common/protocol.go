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
// [ size (2 bytes) ][ data (size bytes) ]
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

// WaitString Receive a string from the socket
// First, reads 2 bytes (big endian) that indicate the length, then reads the payload of that length
// Returns the decoded UTF-8 string or an error if fails
func (p *Protocol) WaitString() (string, error) {
	bigEndianSize, err := p.recvAll(2)
	if err != nil {
		return "", fmt.Errorf("failed to read string size: %w", err)
	}
	payloadSize := uint16(bigEndianSize[0])<<8 | uint16(bigEndianSize[1])

	buf, err := p.recvAll(int(payloadSize))
	if err != nil {
		return "", fmt.Errorf("failed to read string payload: %w", err)
	}
	return string(buf), nil
}

// SendUint8 Send a single unsigned byte (0-255) through the socket
// Returns error if fails
func (p *Protocol) SendUint8(n uint8) error {
	buf := []byte{n}
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

// recvAll Read exactly n bytes from the socket
// Retries until the requested number of bytes is received
// Returns an error if the connection is closed unexpectedly
func (p *Protocol) recvAll(n int) ([]byte, error) {
	msg := make([]byte, n)
	lenMsg := 0

	for lenMsg < n {
		read, err := p.conn.Read(msg[lenMsg:])
		if err != nil {
			return nil, err
		}
		if read == 0 {
			return nil, fmt.Errorf("BrokenPipeError")
		}
		lenMsg += read
	}
	return msg, nil
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
