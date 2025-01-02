package config

import (
	"encoding/json"
	"log"
	"os"
)

const configFileName = ".gatorconfig.json"

type Config struct{
	Db_url				string
	Current_user_name	string
}

func Read() Config{
	path, err := getConfigFilePath()
	if err != nil {
		log.Fatalf("error reading file at path %s", path)
		return Config{}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("error reading file at path %s", path)
		return Config{}
	}
	

	var config Config
	if err := json.Unmarshal([]byte(data), &config); err != nil {
        log.Fatalf("error unmarshalling JSON: %v", err)
		return Config{}
    }
	return config
}

func (c Config) SetUser(user string) error {
	c.Current_user_name = user
	if err := write(c); err != nil {
		return err
	}
	return nil
}

func getConfigFilePath() (string, error) {
	path, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("error getting config file path")
		return "", err
	}
	path += "/" + configFileName
	return path, nil
}

func write(cfg Config) error {
	path, err := getConfigFilePath()
	if err != nil {
		log.Fatalf("error reading file at path %s", path)
		return err
	}

	jsonData, err := json.Marshal(cfg)
	if err != nil {
        log.Fatalf("error unmarshalling JSON: %v", err)
		return err
    }

	if err := os.WriteFile(path, jsonData, 0666); err != nil {
        log.Fatalf("error writing to file %s: %v", path, err)
	}

	return nil
}