package main

import (
	"fmt"
	"io"
	"os"
	"strconv"

	"gopkg.in/yaml.v2"
)

/* Environment utility */

func loadEnvStr(key string, result *string) {
	s, ok := os.LookupEnv(key)
	if !ok {
		return
	}

	*result = s
}

func loadEnvUint(key string, result *uint) {
	s, ok := os.LookupEnv(key)
	if !ok {
		return
	}

	n, err := strconv.Atoi(s)

	if err != nil {
		return
	}

	*result = uint(n) // will clamp the negative value
}

/* Configuration */

type dbConfig struct {
	ConnectionString string `yaml:"connection_string" json:"connection_string"`
}

func defaultDBConfig() dbConfig {
	return dbConfig{
		ConnectionString: "postgres://postgres:postgres@localhost:5432/qiscus_test",
	}
}

func (dc *dbConfig) loadFromEnv() {
	loadEnvStr("QT_DB_CONNECTION_STRING", &dc.ConnectionString)

}

type rdbConfig struct {
	Url string `yaml:"url" json:"url"`
}

func defaultRedisConfig() rdbConfig {
	return rdbConfig{
		Url: "postgres://postgres:postgres@localhost:5432/qiscus_test",
	}
}

func (dc *rdbConfig) loadFromEnv() {
	loadEnvStr("QT_REDIS_URL", &dc.Url)

}

type whConfig struct {
	BaseUrl            string `yaml:"base_url" json:"base_url"`
	MaxCurrentCustomer uint   `yaml:"max_current_customer" json:"max_current_customer"`
}

func defaultWebhookConfig() whConfig {
	return whConfig{
		BaseUrl:            "localhost:3000",
		MaxCurrentCustomer: 3,
	}
}

func (wc *whConfig) loadFromEnv() {
	loadEnvStr("QT_WEBHOOK_BASE_URL", &wc.BaseUrl)
	loadEnvUint("QT_WEBHOOK_MAX_CURRENT_CUSTOMER", &wc.MaxCurrentCustomer)

}

type listenConfig struct {
	Port uint `yaml:"port" json:"port"`
}

func (l listenConfig) Addr() string {
	return fmt.Sprintf(":%d", l.Port)
}

func defaultListenConfig() listenConfig {
	return listenConfig{
		Port: 3000,
	}
}

func (l *listenConfig) loadFromEnv() {
	loadEnvUint("QT_LISTEN_PORT", &l.Port)
}

type qiscusConfig struct {
	BaseUrl   string `yaml:"base_url" json:"base_url"`
	AppID     string `yaml:"app_id" json:"app_id"`
	SecretKey string `yaml:"secret_key" json:"secret_key"`
	Email     string `yaml:"email" json:"email"`
	Password  string `yaml:"password" json:"password"`
	ChannelID uint   `yaml:"channel_id" json:"channel_id"`
}

func defaultQiscusConfig() qiscusConfig {
	return qiscusConfig{
		BaseUrl:   "https://omnichannel.qiscus.com",
		AppID:     "",
		SecretKey: "",
		Email:     "",
		Password:  "",
		ChannelID: 0,
	}
}

func (qc *qiscusConfig) loadFromEnv() {
	loadEnvStr("QT_QISCUS_BASE_URL", &qc.BaseUrl)
	loadEnvStr("QT_QISCUS_APP_ID", &qc.AppID)
	loadEnvStr("QT_QISCUS_SECRET_KEY", &qc.SecretKey)
	loadEnvStr("QT_QISCUS_EMAIL", &qc.Email)
	loadEnvStr("QT_QISCUS_PASSWORD", &qc.Password)
	loadEnvUint("QT_QISCUS_ChannelID", &qc.ChannelID)
}

type config struct {
	Listen        listenConfig `yaml:"listen" json:"listen"`
	DBConfig      dbConfig     `yaml:"db" json:"db"`
	RedisConfig   rdbConfig    `yaml:"redis" json:"redis"`
	QiscusConfig  qiscusConfig `yaml:"qiscus" json:"qiscus"`
	WebhookConfig whConfig     `yaml:"webhook" json:"webhook"`
}

func (c *config) loadFromEnv() {
	c.Listen.loadFromEnv()
	c.DBConfig.loadFromEnv()
	c.RedisConfig.loadFromEnv()
	c.QiscusConfig.loadFromEnv()
	c.WebhookConfig.loadFromEnv()
}

func defaultConfig() config {
	return config{
		Listen:        defaultListenConfig(),
		DBConfig:      defaultDBConfig(),
		RedisConfig:   defaultRedisConfig(),
		QiscusConfig:  defaultQiscusConfig(),
		WebhookConfig: defaultWebhookConfig(),
	}
}

func loadConfigFromReader(r io.Reader, c *config) error {
	return yaml.NewDecoder(r).Decode(c)
}

func loadConfigFromFile(fn string, c *config) error {
	_, err := os.Stat(fn)

	if err != nil {
		return err
	}

	f, err := os.Open(fn)

	if err != nil {
		return err
	}

	defer f.Close()

	return loadConfigFromReader(f, c)
}

/* How to load the configuration, the highest priority loaded last
 * First: Initialise to default config
 * Second: Replace with environment variables
 * Third: Replace with configuration file
 */

func loadConfig(fn string) config {
	cfg := defaultConfig()
	cfg.loadFromEnv()

	loadConfigFromFile(fn, &cfg)

	return cfg
}
