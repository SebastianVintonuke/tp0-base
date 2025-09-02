package common

import (
	"bufio"
	"net"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/op/go-logging"
)

// BATCH_MAX_SIZE default value so that packets do not exceed 8kB
// 1 bet = 61 bytes = (2 bytes * 6 fields) + aprox 49 bytes
// 8192 bytes / 61 = aprox 135 bets
const BATCH_MAX_SIZE = 135

const (
	ErrorCode               = 0x00
	AckCode                 = 0xFF
	OperationCodeUploadBets = 0x01
	OperationCodeGetWinners = 0x02
)

var log = logging.MustGetLogger("log")

// ClientConfig Configuration used by the client
type ClientConfig struct {
	ID             string
	ServerAddress  string
	BatchMaxAmount int
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
	file       *os.File
	protocol   *Protocol
	wasStopped uint32
}

// NewClient Initializes a new client receiving the configuration
// as a parameter
func NewClient(config ClientConfig, file string) (*Client, error) {
	fd, err := os.Open(file)
	if err != nil {
		log.Criticalf("action: open_file | result: fail | client_id: %v | error: %v",
			config.ID,
			err,
		)
		return nil, err
	}
	client := &Client{
		config: config,
		file:   fd,
	}
	return client, nil
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

// StartClient Starts the client
// First upload bets to the server in batches,
// then wait for server acknowledgement,
// then try to get winners until success,
// finally shutdown gracefully
// If an unexpected error occurs shutdown gracefully
func (c *Client) StartClient() {
	c.createClientSocket()
	err := c.protocol.SendUint8(uint8(OperationCodeUploadBets))
	if err != nil {
		c.GracefulShutdown()
		return
	}
	ack, err := c.waitAck()
	if err != nil || ack != AckCode {
		c.GracefulShutdown()
		return
	}
	c.OperationUpload()
	c.GracefulShutdown()
	for {
		c.createClientSocket()
		err = c.protocol.SendUint8(uint8(OperationCodeGetWinners))
		if err != nil {
			c.GracefulShutdown()
			return
		}
		ack, err = c.waitAck()
		if err != nil {
			c.GracefulShutdown()
			return
		}
		if ack == AckCode {
			c.OperationGetWinner()
			break
		}
		c.GracefulShutdown()
		time.Sleep(1 * time.Second)
	}

	c.GracefulShutdown()
}

// GracefulShutdown Gracefully shutdown the server
// Close sockets and file descriptors
func (c *Client) GracefulShutdown() {
	log.Infof("action: graceful_shutdown | result: in_progress | client_id: %v", c.config.ID)

	c.setWasStopped()
	if c.protocol != nil {
		c.protocol.CloseWith(c.tryClose)
	}

	if c.file != nil {
		err := c.file.Close()
		if err != nil {
			log.Errorf("action: close_file | result: fail | client_id: %v | error: %v",
				c.config.ID,
				err,
			)
		}
		c.file = nil
	}

	log.Infof("action: exit | result: success | client_id: %v", c.config.ID)
}

// OperationUpload Reads bets from the input file in batches and sends them to the server,
// waits for acknowledgements after each batch, logs success or errors,
// stops when all bets are sent or in an error
func (c *Client) OperationUpload() {
	reader := bufio.NewReader(c.file)
	totalBets := 0

	for {
		batch, err := c.readNextBatch(reader)
		if err != nil && err.Error() != "EOF" {
			log.Errorf("action: read_file | result: fail | client_id: %v | error: %v",
				c.config.ID,
				err,
			)
			break
		}

		err = c.sendBatch(batch)
		if err != nil {
			log.Errorf("action: apuesta_enviada | result: fail | client_id: %v | error: %v",
				c.config.ID,
				err,
			)
			return
		}

		log.Infof("action: apuesta_enviada | result: success | cantidad: %v", len(batch))

		ack, err := c.waitAck()
		if err != nil || ack != AckCode {
			log.Errorf("action: apuesta_enviada | result: fail | client_id: %v | error: %v",
				c.config.ID,
				err,
			)
			return
		}

		totalBets += len(batch)
		if len(batch) == 0 {
			log.Infof("action: apuestas totales | result: success | cantidad: %v", totalBets)
			break
		}
	}
}

// OperationGetWinner Requests winners from the server for this client's agency
// First send the agency ID, then waits for the winners
// Logs the number of winners or errors if error
func (c *Client) OperationGetWinner() {
	err := c.protocol.SendString(c.config.ID)
	if err != nil {
		log.Errorf("action: send_agency | result: fail | client_id: %v | error: %v",
			c.config.ID, err)
		return
	}
	winners, err := c.waitWinners()
	if err != nil {
		log.Errorf("action: consulta_ganadores | result: fail | client_id: %v | error: %v",
			c.config.ID, err)
		return
	}
	log.Infof("action: consulta_ganadores | result: success | cant_ganadores: %v",
		len(winners),
	)
}

// sendBet Sends the client ID and a bet using the protocol
// Returns an error if fails
//
// [ ID (aprox 3 bytes) ][ FirstName (aprox 12 bytes) ][ LastName (aprox 12 bytes) ]
// [ Document (8 bytes) ][ Birthdate (10 bytes) ][ Number (aprox 4 bytes) ]
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

// sendBatch Sends a batch of bets to the server
// First sends the size of the batch as an uint8, followed by the bets
// Returns an error if any fails
//
// [ n bets (1 byte) ][ bet 1 ][ bet 2 ][ ... ][ bet n ]
func (c *Client) sendBatch(bets []Bet) error {
	if err := c.protocol.SendUint8(uint8(len(bets))); err != nil {
		return err
	}
	for _, bet := range bets {
		err := c.sendBet(bet)
		if err != nil {
			return err
		}
	}
	return nil
}

// waitWinners Waits for the list of winners from the server
// Returns the List of winning documents or an error
func (c *Client) waitWinners() ([]string, error) {
	lenWinners, err := c.protocol.WaitUint8()
	if err != nil {
		return nil, err
	}

	winners := make([]string, 0, lenWinners)
	for i := 0; i < int(lenWinners); i++ {
		winner, err := c.protocol.WaitString()
		if err != nil {
			return nil, err
		}
		winners = append(winners, winner)
	}

	return winners, nil
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

// readNextBet Reads the next line from the input file and parses it into a Bet
// Returns a Bet or an error
func (c *Client) readNextBet(reader *bufio.Reader) (Bet, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return Bet{}, err
	}

	line = strings.TrimSpace(line)
	fields := strings.Split(line, ",")

	return Bet{
		FirstName: fields[0],
		LastName:  fields[1],
		Document:  fields[2],
		Birthdate: fields[3],
		Number:    fields[4],
	}, nil
}

// readNextBatch Reads up to BatchMaxAmount bets from the file
// Accumulates bets until reaching the limit or EOF
// Returns the batch, the batch and EOF error or an error
func (c *Client) readNextBatch(reader *bufio.Reader) ([]Bet, error) {
	var batchSize int
	if BATCH_MAX_SIZE < c.config.BatchMaxAmount {
		batchSize = BATCH_MAX_SIZE
	} else {
		batchSize = c.config.BatchMaxAmount
	}

	bets := make([]Bet, 0, batchSize)
	for len(bets) < batchSize {
		bet, err := c.readNextBet(reader)
		if err != nil {
			if err.Error() == "EOF" {
				return bets, err
			}
			return nil, err
		}
		bets = append(bets, bet)
	}

	return bets, nil
}
