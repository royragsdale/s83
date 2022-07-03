package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"regexp"

	"github.com/royragsdale/s83"
)

const defaultConfigName = "default"

const blankConfig = `public =
secret =
server =`

type Config struct {
	Name      string
	Creator   s83.Creator
	Server    *url.URL
	Follows   []s83.Follow
	templates *template.Template
	Favicon   string
}

func configDir() string {
	configRoot, err := os.UserConfigDir()
	if err != nil {
		log.Fatal("Error finding config directory: ", err)
	}
	return filepath.Join(configRoot, "s83")
}

func configPath(name string) string {
	return filepath.Join(configDir(), name)
}

func dataPath(name string) string {
	return filepath.Join(configDir(), "data", name)
}

func (c Config) Path() string {
	return configPath(c.Name)
}

func (c Config) DataPath() string {
	return dataPath(c.Name)
}

func initConfig(name string) []byte {
	configDir := configDir()
	configPath := configPath(name)

	err := os.MkdirAll(configDir, 0700)
	if err != nil {
		log.Fatalf("Error creating config directory: %s : %v", configDir, err)
	}

	_, err = os.ReadFile(configPath)
	if !errors.Is(err, os.ErrNotExist) {
		log.Fatalf("Error: config (%s) already exists refusing to clobber", configPath)
	}

	cFile, err := os.Create(configPath)
	if err != nil {
		log.Fatalf("Error creating config file: %s : %v", configPath, err)
	}
	defer cFile.Close()

	// set mode to User R/W since it will contain a private key
	err = cFile.Chmod(0600)
	if err != nil {
		log.Fatalf("Error setting permissions on initial config: %v", err)
	}

	_, err = cFile.Write([]byte(blankConfig))
	if err != nil {
		log.Fatalf("Error storing initial config: %v", err)
	}
	cFile.Close()

	// confirm we can re-read the file
	data, err := os.ReadFile(configPath)
	if err != nil {
		log.Fatalf("Error reading initial config: %v", err)
	}

	return data
}

func parseConfig(data []byte, name string) Config {
	config := Config{}
	config.Name = name

	// match configuration keys (secert=, server=)
	rePrivateKey := regexp.MustCompile(`(?m)^secret\s*=\s*([0-9A-Fa-f]{64}?)$`)
	reServer := regexp.MustCompile(`(?m)^server\s*=\s*(.*)$`)

	serverMatch := reServer.FindSubmatch(data)
	if serverMatch != nil && len(serverMatch) == 2 {
		serverStr := string(serverMatch[1])
		url, err := url.Parse(serverStr)
		if err != nil {
			fmt.Printf("[warn] Invalid server configuration: %v\n", err)
		} else if url.Scheme != "http" && url.Scheme != "https" && serverStr != "" {
			fmt.Printf("[warn] Invalid server configuration: must include `http` or `https` (%s)\n", url)
		} else {
			config.Server = url
		}
	}

	privateKeyHexMatch := rePrivateKey.FindSubmatch(data)
	if privateKeyHexMatch != nil && len(privateKeyHexMatch) == 2 {
		pkHex := string(privateKeyHexMatch[1])
		creator, err := s83.NewCreatorFromKey(pkHex)
		if err != nil {
			fmt.Printf("[warn] Invalid secret configuration: %v\n", err)
		} else {
			config.Creator = creator
		}
	}
	config.Follows = s83.ParseSpringfileFollows(data)

	// load templates
	config.templates = template.Must(template.ParseFS(resources, "templates/*.tmpl"))
	if config.templates == nil {
		log.Fatal("Error loading templates")
	}

	config.Favicon = base64.StdEncoding.EncodeToString(favicon)

	return config
}

func loadConfig(name string) Config {
	configPath := configPath(name)

	data, err := os.ReadFile(configPath)
	if errors.Is(err, os.ErrNotExist) {
		fmt.Printf("[info] Config did not exist. Initializing a config at %s\n", configPath)
		data = initConfig(name)

	} else if err != nil {
		log.Fatalf("Error loading config: %v\n", err)
	}

	// check permissions
	fi, err := os.Stat(configPath)
	if err != nil {
		log.Fatalf("Error checking permissions on config: %v\n", err)
	} else if fi.Mode() != 0600 {
		log.Fatal("Insecure configuration must be 0600 to protect your private key")
	}

	// ensure config data dir exists
	dataPath := dataPath(name)
	err = os.MkdirAll(dataPath, 0700)
	if err != nil {
		log.Fatalf("Error creating config data directory: %s : %v", dataPath, err)
	}

	return parseConfig(data, name)
}

func (config Config) String() string {
	display := fmt.Sprintf("name    : %s\n", config.Name)
	display += fmt.Sprintf("path    : %s\n", config.Path())
	display += fmt.Sprintf("server  : %s\n", config.Server)
	display += fmt.Sprintf("pub     : %s\n", config.Creator)
	display += fmt.Sprintf("---------\nfollows :\n")
	for _, follow := range config.Follows {
		display += fmt.Sprintf("%s\n", follow)
	}

	return display
}
