// Package configs contains the system configurations.
package configs

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
)

type configData struct {
	ServerPort     int32  `json:"port"`
	DatabaseDSN    string `json:"database_dsn"`
	DatabaseDriver string `json:"database_driver"`
	PrivateKeyFile string `json:"private_key_file"`
}

// Config holds the system configuration.
type Config interface {
	ServerPort() int32
	DatabaseDSN() string
	DatabaseDriver() string
	PrivateKeyFile() string
	PrivateKey() rsa.PrivateKey
}

type defaultConfig struct {
	data       *configData
	privateKey *rsa.PrivateKey
}

func (c *defaultConfig) ServerPort() int32 {
	return c.data.ServerPort
}

func (c *defaultConfig) DatabaseDSN() string {
	return c.data.DatabaseDSN
}

func (c *defaultConfig) DatabaseDriver() string {
	return c.data.DatabaseDriver
}

func (c *defaultConfig) PrivateKeyFile() string {
	return c.data.PrivateKeyFile
}

func (c *defaultConfig) PrivateKey() rsa.PrivateKey {
	return *c.privateKey
}

func (c *defaultConfig) loadPrivateKey(configPath string) error {
	path := c.PrivateKeyFile()
	if _, err := os.Stat(c.PrivateKeyFile()); os.IsNotExist(err) {
		path = fmt.Sprintf("%s/%s", configPath, path)
	}
	pemFile, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	privatePem, _ := pem.Decode(pemFile)
	var parsedKey interface{}
	parsedKey, err = x509.ParsePKCS1PrivateKey(privatePem.Bytes)
	if err != nil {
		return err
	}
	pk, isPrivateKey := parsedKey.(*rsa.PrivateKey)
	if !isPrivateKey {
		return errors.New("the given private key is not valid")
	}
	c.privateKey = pk
	return nil
}

// Load loads the given configuration file.
func Load(configPath string) (Config, error) {
	data := &configData{}
	configFile, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("an occurred while loading config file: %w", err)
	}
	err = json.NewDecoder(configFile).Decode(data)
	if err != nil {
		return nil, fmt.Errorf("an occurred while parsing config file: %w", err)
	}
	configuration := &defaultConfig{data: data}
	if configuration.PrivateKeyFile() != "" {
		if err = configuration.loadPrivateKey(configPath); err != nil {
			return nil, err
		}
	}
	return configuration, nil
}

// MustLoad loads the given configuration file and if any error occurs, will panic.
func MustLoad(configPath string) Config {
	config, err := Load(configPath)
	if err != nil {
		panic(err)
	}
	return config
}

