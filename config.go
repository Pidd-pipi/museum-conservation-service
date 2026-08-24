package main

import (
	"os"

	"museum-conservation-service/domain"
)

type Config struct {
	Port string
}

var serviceLabel = domain.ServiceName

func LoadConfig() Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	return Config{Port: port}
}
