package mpclient

import (
	"fmt"
	"io"
	"io/ioutil"

	"github.com/aws/aws-sdk-go/aws/credentials"
	v4 "github.com/aws/aws-sdk-go/aws/signer/v4"
	"github.com/caarlos0/env"
)

const (
	authService = "cf"
	authRegion  = "eu-west-1"
)

// MultiplayClient represents something capable of interfacing with the multiplay API
type MultiplayClient interface {
	Allocate(fleet, region string, profile int64, uuid string) (*AllocateResponse, error)
	Allocations(fleet, region string, profile int64, uuid string) ([]AllocationResponse, error)
	Deallocate(fleet, uuid string) error
}

// Config holds configuration used to access the multiplay api
type Config struct {
	AccessKey string `env:"MP_ACCESS_KEY"`
	SecretKey string `env:"MP_SECRET_KEY"`
}

// multiplayClient is the implementation of the multiplay client
type multiplayClient struct {
	signer  *v4.Signer
	baseURL string
}

// NewClientFromEnv creates a multiplay client from the environment
func NewClientFromEnv() (MultiplayClient, error) {
	cfg := Config{}
	if err := env.Parse(&cfg); err != nil {
		return nil, fmt.Errorf("failed to load multiplay config from env: %w", err)
	}

	if cfg.AccessKey == "" {
		return nil, fmt.Errorf("access key is empty")
	}

	if cfg.SecretKey == "" {
		return nil, fmt.Errorf("access key is empty")
	}

	return NewClient(cfg), nil
}

// NewClient creates a multiplay client
func NewClient(cfg Config) MultiplayClient {
	c := &multiplayClient{}
	c.signer = v4.NewSigner(credentials.NewStaticCredentials(cfg.AccessKey, cfg.SecretKey, ""))

	c.baseURL = "https://api-dev.multiplay.co.uk"
	return c
}

func getBody(w io.Reader) string {
	v, err := ioutil.ReadAll(w)
	if err != nil {
		return ""
	}
	return string(v)
}
