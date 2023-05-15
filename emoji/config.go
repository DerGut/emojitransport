package emoji

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

type Config struct {
	Directory string `json:"directory"`

	Slack struct {
		Token  string `json:"token"`
		Route  string `json:"route"`
		Cookie string `json:"cookie"`
	} `json:"slack"`

	Confluence struct {
		Token string `json:"token"`
	} `json:"confluence"`
}

func ParseConfig(path string) (Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return Config{}, fmt.Errorf("open file: %w", err)
	}

	content, err := io.ReadAll(file)
	if err != nil {
		return Config{}, fmt.Errorf("read config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(content, &config); err != nil {
		return Config{}, fmt.Errorf("unmarshal config: %w", err)
	}

	return config, nil
}
