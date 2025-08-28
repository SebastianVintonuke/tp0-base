package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/op/go-logging"
	"github.com/pkg/errors"
	"github.com/spf13/viper"

	"github.com/7574-sistemas-distribuidos/docker-compose-init/client/common"
)

var log = logging.MustGetLogger("log")

// InitConfig Function that uses viper library to parse configuration parameters.
// Viper is configured to read variables from both environment variables and the
// config file ./config.yaml. Environment variables takes precedence over parameters
// defined in the configuration file. If some of the variables cannot be parsed,
// an error is returned
func InitConfig() (*viper.Viper, error) {
	v := viper.New()

	// Configure viper to read env variables with the CLI_ prefix
	v.AutomaticEnv()
	v.SetEnvPrefix("cli")
	// Use a replacer to replace env variables underscores with points. This let us
	// use nested configurations in the config file and at the same time define
	// env variables for the nested configurations
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Add env variables supported
	v.BindEnv("id")
	v.BindEnv("server", "address")
	v.BindEnv("log", "level")

	// Try to read configuration from config file. If config file
	// does not exists then ReadInConfig will fail but configuration
	// can be loaded from the environment variables so we shouldn't
	// return an error in that case
	v.SetConfigFile("./config.yaml")
	if err := v.ReadInConfig(); err != nil {
		fmt.Printf("Configuration could not be read from config file. Using env variables instead")
	}

	return v, nil
}

// InitLogger Receives the log level to be set in go-logging as a string. This method
// parses the string and set the level to the logger. If the level string is not
// valid an error is returned
func InitLogger(logLevel string) error {
	baseBackend := logging.NewLogBackend(os.Stdout, "", 0)
	format := logging.MustStringFormatter(
		`%{time:2006-01-02 15:04:05} %{level:.5s}     %{message}`,
	)
	backendFormatter := logging.NewBackendFormatter(baseBackend, format)

	backendLeveled := logging.AddModuleLevel(backendFormatter)
	logLevelCode, err := logging.LogLevel(logLevel)
	if err != nil {
		return err
	}
	backendLeveled.SetLevel(logLevelCode, "")

	// Set the backends to be used.
	logging.SetBackend(backendLeveled)
	return nil
}

// PrintConfig Print all the configuration parameters of the program.
// For debugging purposes only
func PrintConfig(v *viper.Viper) {
	log.Infof("action: config | result: success | client_id: %s | server_address: %s | log_level: %s",
		v.GetString("id"),
		v.GetString("server.address"),
		v.GetString("log.level"),
	)
}

// InitBet Function that uses viper library to parse bet parameters.
func InitBet() (*viper.Viper, error) {
	v := viper.New()

	v.AutomaticEnv()

	keys := []string{"nombre", "apellido", "documento", "nacimiento", "numero"}
	for _, key := range keys {
		err := v.BindEnv(key)
		if err != nil {
			return v, errors.Wrapf(err, "Bet %s could not be read from env variables", key)
		}
	}

	return v, nil
}

// PrintBet Print all the bet parameters of the program.
// For debugging purposes only
func PrintBet(v *viper.Viper) {
	log.Infof(
		"action: bet | result: success | first name: %s | last name: %s | document: %s | birthdate: %s | number: %s",
		v.GetString("nombre"),
		v.GetString("apellido"),
		v.GetString("documento"),
		v.GetString("nacimiento"),
		v.GetString("numero"),
	)
}

// InitSignalHandler Registers a signal handler for SIGTERM
func InitSignalHandler(handler func()) {
	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, syscall.SIGTERM)
	go func() {
		<-signalChannel
		handler()
	}()
}

func main() {
	v, err := InitConfig()
	if err != nil {
		log.Criticalf("%s", err)
	}

	if err := InitLogger(v.GetString("log.level")); err != nil {
		log.Criticalf("%s", err)
	}

	// Print program config with debugging purposes
	PrintConfig(v)

	clientConfig := common.ClientConfig{
		ServerAddress: v.GetString("server.address"),
		ID:            v.GetString("id"),
	}

	bet, err := InitBet()
	if err != nil {
		log.Criticalf("%s", err)
	}

	// Print program bet with debugging purposes
	PrintBet(bet)

	clientBet := common.Bet{
		FirstName: bet.GetString("nombre"),
		LastName:  bet.GetString("apellido"),
		Document:  bet.GetString("documento"),
		Birthdate: bet.GetString("nacimiento"),
		Number:    bet.GetString("numero"),
	}

	client := common.NewClient(clientConfig, clientBet)
	InitSignalHandler(client.GracefulShutdown)
	client.StartClient()
}
