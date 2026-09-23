package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type config struct {
	BaseURL string `json:"base_url"`
	Token   string `json:"token"`
}

func configPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "koffan-tui", "config.json"), nil
}

func loadConfig() (config, string, error) {
	path, err := configPath()
	if err != nil {
		return config{}, "", err
	}
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			return config{}, path, err
		}
		template := []byte("{\n  \"base_url\": \"http://localhost:3000\",\n  \"token\": \"\"\n}\n")
		if err := os.WriteFile(path, template, 0o600); err != nil {
			return config{}, path, err
		}
		return config{}, path, fmt.Errorf("fichier créé; renseignez le token puis relancez")
	}
	if err != nil {
		return config{}, path, err
	}
	var cfg config
	if err := json.Unmarshal(b, &cfg); err != nil {
		return config{}, path, err
	}
	cfg.BaseURL = strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	cfg.Token = strings.TrimSpace(cfg.Token)
	if cfg.BaseURL == "" || cfg.Token == "" {
		return config{}, path, errors.New("base_url et token sont obligatoires")
	}
	return cfg, path, nil
}
