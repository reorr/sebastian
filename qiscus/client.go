package qiscus

import (
	"log/slog"
	"net/http"

	"github.com/redis/go-redis/v9"
)

// Config holds the Qiscus API configuration
type Config struct {
	BaseUrl     string
	AppID       string
	SecretKey   string
	Email       string
	Password    string
	ChannelID   string
}

// Client wraps the HTTP client and configuration for Qiscus API
type Client struct {
	config     *Config
	httpClient *http.Client
	logger     *slog.Logger
	redis      *redis.Client
}

// NewClient creates a new Qiscus API client
func NewClient(config *Config, logger *slog.Logger, redisClient *redis.Client) *Client {
	return &Client{
		config:     config,
		httpClient: &http.Client{},
		logger:     logger,
		redis:      redisClient,
	}
}

// NewClientWithHTTPClient creates a new Qiscus API client with custom HTTP client
func NewClientWithHTTPClient(config *Config, httpClient *http.Client, logger *slog.Logger, redisClient *redis.Client) *Client {
	return &Client{
		config:     config,
		httpClient: httpClient,
		logger:     logger,
		redis:      redisClient,
	}
}

// GetConfig returns the client configuration
func (c *Client) GetConfig() *Config {
	return c.config
}

// GetHTTPClient returns the HTTP client
func (c *Client) GetHTTPClient() *http.Client {
	return c.httpClient
}

// GetLogger returns the logger
func (c *Client) GetLogger() *slog.Logger {
	return c.logger
}

// GetRedis returns the Redis client
func (c *Client) GetRedis() *redis.Client {
	return c.redis
}