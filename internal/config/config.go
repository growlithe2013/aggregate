package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const configFileName = ".gatorconfig.json"

type Config struct {
	Db_url            string `json:"db_url"`
	Current_user_name string `json:"current_user_name"`
}

func Read() *Config {
	var cfg *Config
	path, err := getConfigFilePath()
	if err != nil {
		fmt.Println("Error reading config file:", err)
		return cfg
	}
	dat, err := os.ReadFile(path)
	if err != nil {
		fmt.Println("Error reading config file:", err)
		return cfg
	}

	err = json.Unmarshal(dat, &cfg)
	if err != nil {
		fmt.Println("Error parsing config file:", err)
		return cfg
	}
	return cfg
}

func (cfg *Config) SetUser(user string) error {
	cfg.Current_user_name = user
	err := write(*cfg)
	return err
}

func getConfigFilePath() (string, error) {
	homedir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(homedir, configFileName)
	return path, nil
}

func write(cfg Config) error {
	path, err := getConfigFilePath()
	if err != nil {
		fmt.Println("Error reading config file:", err)
		return err
	}
	dat, err := json.Marshal(cfg)
	if err != nil {
		fmt.Println("Error parsing config file:", err)
		return err
	}
	err = os.WriteFile(path, dat, 0644)
	if err != nil {
		fmt.Println("Error writing config file:", err)
		return err
	}
	return nil
}
