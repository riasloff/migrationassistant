package configreader

import (
	"encoding/json"
	"io"
	"log"
	"os"
)

type Config struct {
	Database []Database `json:"database"`
	Server   Server     `json:"server"`
}

type Database struct {
	Driver string `json:"driver"`
	Dsn    string `json:"dsn"`
}

type Server struct {
	Port string `json:"port"`
}

var cfg Config

func ConfigReader(filepath string) (Config, error) {
	log.Printf("reading config from path: %s\n", filepath)

	jsonFile, err := os.Open(filepath)
	if err != nil {
		log.Printf("failed to open json file: %s, error: %v", filepath, err)
		return cfg, err
	}
	defer jsonFile.Close()

	jsonData, err := io.ReadAll(jsonFile)
	if err != nil {
		log.Printf("failed to read json file, error: %v", err)
		return cfg, err
	}

	if err := json.Unmarshal(jsonData, &cfg); err != nil {
		log.Printf("failed to unmarshal json file, error: %v", err)
		return cfg, err
	}

	return cfg, nil
}
