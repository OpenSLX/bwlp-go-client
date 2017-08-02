package client

import (
	"fmt"
	"log"
	"sync"
	"crypto/tls"
	"errors"
	"git.apache.org/thrift.git/lib/go/thrift"
	"github.com/OpenSLX/bwlp-go-client/bwlp"
)

type MasterServerEndpoint struct {
	Hostname string
	PortSSL int
	PortPlain int
}

var (
	// endpoint to the bwlp masterserver
	endpoint *MasterServerEndpoint
	// singleton client instance
	masterClient *bwlp.MasterServerClient
	// thread-safe function executor
	once sync.Once
)

// Initialize the masterserver client using the server's
// expected transport (framed) and protocol (binary).
// Enforces the use of SSL for now.
func initClient(addr string) error {
	var transport thrift.TTransport
	var err error
	cfg := &tls.Config{
		MinVersion:	tls.VersionTLS12,
		CurvePreferences:	[]tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		CipherSuites:	[]uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
			tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		},
		PreferServerCipherSuites: true,
	}
	transport, err = thrift.NewTSSLSocket(addr, cfg)
	if err != nil {
		log.Printf("Error opening SSL socket: %s\n", err)
		return err
	}
	// framed transport is required
	transportFactory := thrift.NewTFramedTransportFactory(thrift.NewTTransportFactory())
	transport = transportFactory.GetTransport(transport)
	if err := transport.Open(); err != nil {
		log.Println("Error opening transport layer for reading/writing: %s\n", err)
		return err
	}
	// binary proto is required
	protocolFactory := thrift.NewTBinaryProtocolFactoryDefault()

	// now retrieve a new client and test it
	masterClient = nil
	client := bwlp.NewMasterServerClientFactory(transport, protocolFactory)
	if client == nil {
		return fmt.Errorf("Thrift client factory return nil client!")
	}
	if _, err := client.Ping(); err != nil {
		log.Printf("Error pinging masterserver: %s\n", err)
		return err
  }
	log.Printf("## Connection established to: %s ##", addr)
	masterClient = client
	return nil
}

// Global setter for the endpoint
func SetEndpoint(param *MasterServerEndpoint) error {
	if param == nil {
		return errors.New("Invalid endpoint given!")
	}
	if masterClient != nil {
		return errors.New("MasterServer client is already initialized!")
	}
	// TODO user-supplied endpoints should be validated abit
	endpoint = param
	return nil
}

// Global access to the singleton client instance
func GetInstance() (client *bwlp.MasterServerClient) {
	// check that endpoint was set
	if endpoint == nil {
		log.Printf("No endpoint set! Set one first.\n")
		return
	}
	// initialize the client only once, in essence
	// a simple kind of singleton pattern
	once.Do(func() {
		masterServerAddress := fmt.Sprintf("%s:%d", endpoint.Hostname, endpoint.PortSSL)
		if err := initClient(masterServerAddress); err != nil {
			log.Printf("Error initialising client: %s\n", err)
		}
	})
	return masterClient
}
