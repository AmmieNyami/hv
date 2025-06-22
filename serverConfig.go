package main

import (
	"fmt"
	"os"
	"path"
)

type ServerConfig struct {
	FrontendURL  string `json:"frontend_url"`
	DatabasePath string `json:"database_path"`
	Port         int    `json:"port"`

	DisableRegistering bool `json:"disable_registering"`
}

func LoadServerConfig() ServerConfig {
	findFile := func(file string) string {
		if file[0] == '/' {
			if _, err := os.Stat(file); os.IsNotExist(err) {
				fmt.Fprintf(os.Stderr, "ERROR: cound not find file `%s`\n", file)
				os.Exit(1)
			}
			return file
		}

		exeDirectory, _ := exeDirectory()
		workingDirectory, _ := os.Getwd()

		for _, directory := range []string{workingDirectory, exeDirectory} {
			path := path.Join(directory, file)
			if _, err := os.Stat(path); os.IsNotExist(err) {
				continue
			}
			return path
		}

		fmt.Fprintf(os.Stderr, "ERROR: cound not find file `%s`\n", file)
		os.Exit(1)
		panic("unreachable")
	}

	// Load server config
	serverConfigFilePath := findFile("config.json")
	serverConfigFile, err := os.ReadFile(serverConfigFilePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: failed to open file `%s`: %v\n", serverConfigFilePath, err)
		os.Exit(1)
	}

	var serverConfig ServerConfig
	err = UnmarshalJsonWithComments(string(serverConfigFile), &serverConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: failed to decode JSON file `%s`: %v\n", serverConfigFilePath, err)
		os.Exit(1)
	}

	// Validate fields
	if !isURLValid(serverConfig.FrontendURL) {
		fmt.Fprintf(os.Stderr, "ERROR: invalid frontend URL specified in configuration file\n")
		os.Exit(1)
	}

	databasePath := findFile(serverConfig.DatabasePath)

	if serverConfig.Port == 0 {
		fmt.Fprintf(os.Stderr, "ERROR: invalid port specified in configuration file\n")
		os.Exit(1)
	}

	// Return validated struct
	return ServerConfig{
		FrontendURL:  serverConfig.FrontendURL,
		DatabasePath: databasePath,
		Port:         serverConfig.Port,

		DisableRegistering: serverConfig.DisableRegistering,
	}
}
