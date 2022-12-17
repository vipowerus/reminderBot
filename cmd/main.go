package main

import (
	"flag"
	"fmt"
	"log"
	"reminder/internal/server"

	"github.com/BurntSushi/toml"
)

var (
	configPath string
)

func init() {
	flag.StringVar(&configPath, "config-path", "configs/env.toml", "path to config file")
}

func main() {
	flag.Parse()

	config := server.NewConfig()
	_, err := toml.DecodeFile(configPath, config)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Print(config)
	s := server.New(config)
	if err := s.Start(); err != nil {
		log.Fatal(err)
	}
}
