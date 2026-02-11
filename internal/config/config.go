package config

import (
	"encoding/json"
	"os"
)

type Config struct {
	BiliDir    string `json:"bili_dir"`
	FFmpegPath string `json:"ffmpeg_path"`
}

const ConfigFile = "quell_config.json"

func Load() (*Config, error) {
	data, err := os.ReadFile(ConfigFile)
	if err != nil {
		return nil, err
	}
	var cfg Config
	err = json.Unmarshal(data, &cfg)
	return &cfg, err
}

func Save(cfg Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(ConfigFile, data, 0644)
}
