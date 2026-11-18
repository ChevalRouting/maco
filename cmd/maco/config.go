package main

import "github.com/m-vinc/maco/pkg/config"

type CLIConfig struct {
	LogLevel string
	DataDir  string
	Paths    *config.Paths
}

var cli = &CLIConfig{}
