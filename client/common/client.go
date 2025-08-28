package common

import (
	"net"
	"sync/atomic"

	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("log")

// ClientConfig Configuration used by the client
type ClientConfig struct {
	ID            string
	ServerAddress string
}

// Bet bet used by the client
type Bet struct {
	FirstName string
	LastName  string
	Document  string
	Birthdate string
	Number    string
}

// Client Entity that encapsulates how
type Client struct {
	config     ClientConfig
	bet        Bet
	protocol   *Protocol
	wasStopped uint32
}

// NewClient Initializes a new client receiving the configuration
// as a parameter
func NewClient(config ClientConfig, bet Bet) *Client {
	client := &Client{
		config: config,
		bet:    bet,
	}
	return client
}

// CreateClientSocket Initializes client socket. In case of
// failure, error is printed in stdout/stderr and exit 1
// is returned
func (c *Client) createClientSocket() error {
	conn, err := net.Dial("tcp", c.config.ServerAddress)
	if err != nil {
		log.Criticalf(
			"action: connect | result: fail | client_id: %v | error: %v",
			c.config.ID,
			err,
		)
	}
	c.protocol = NewProtocol(conn)
	return nil
}

func (c *Client) StartClient() {
	c.createClientSocket()

	err := c.sendBet(c.bet)
	if err != nil {
		log.Errorf("action: apuesta_enviada | result: fail | client_id: %v | error: %v",
			c.config.ID,
			err,
		)
		return
	}

	ack, err := c.waitAck()

	if err != nil {
		log.Errorf("action: apuesta_enviada | result: fail | client_id: %v | error: %v",
			c.config.ID,
			err,
		)
		return
	}

	if ack != 0 {
		log.Errorf("action: apuesta_enviada | result: fail | client_id: %v",
			c.config.ID,
		)
		return
	}

	log.Infof("action: apuesta_enviada | result: success | dni: %s | numero: %s",
		c.bet.Document,
		c.bet.Number,
	)

    c.GracefulShutdown()
}

// GracefulShutdown Gracefully shutdown the server
func (c *Client) GracefulShutdown() {
	log.Infof("action: graceful_shutdown | result: in_progress | client_id: %v", c.config.ID)

	c.setWasStopped()
	if c.protocol != nil {
		c.protocol.CloseWith(c.tryClose)
	}

	log.Infof("action: exit | result: success | client_id: %v", c.config.ID)
}

// sendBet Sends the client ID and a bet using the protocol
// Returns an error if fails
func (c *Client) sendBet(bet Bet) error {
	fields := []string{
		c.config.ID,
		bet.FirstName,
		bet.LastName,
		bet.Document,
		bet.Birthdate,
		bet.Number,
	}
	for _, field := range fields {
		if err := c.protocol.SendString(field); err != nil {
			return err
		}
	}
	return nil
}

// waitAck Waits for an acknowledgement byte from the server
// Returns the received uint8 value or an error
func (c *Client) waitAck() (uint8, error) {
	return c.protocol.WaitUint8()
}

// tryClose Attempt to gracefully close a given socket and log the result
func (c *Client) tryClose(aSocket net.Conn) {
	err := aSocket.Close()
	if err != nil {
		log.Errorf("action: close_socket | result: fail | client_id: %v | error: %v",
			c.config.ID,
			err,
		)
		return
	}
	log.Infof("action: close_socket | result: success | client_id: %v",
		c.config.ID,
	)
}

// setWasStopped Marks the client as stopped
func (c *Client) setWasStopped() {
	atomic.StoreUint32(&c.wasStopped, 1)
}

// getWasStopped Returns if the client was stopped
func (c *Client) getWasStopped() bool {
	return atomic.LoadUint32(&c.wasStopped) == 1
}
