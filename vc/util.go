package vc

import (
	"encoding/json"
	"github.com/pkg/errors"
)

func DeepClone(cfg *Config) (*Config, error) {
	data, err := json.Marshal(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "marshalling base config failed")
	}
	cfg = &Config{}
	err = json.Unmarshal(data, cfg)
	if err != nil {
		return nil, errors.Wrap(err, "rebuilding config failed")
	}
	return cfg, nil
}
