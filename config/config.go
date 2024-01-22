package config

import "github.com/pelletier/go-toml"

const (
	DefaultConfigPath = "config.toml"
)

type RPCNode struct {
	Name        string
	RPCEndpoint string `toml:"rpc"`
	WSEndpoint  string `toml:"ws"`
	Observer    bool   `toml:"observer"`
}

type Config struct {
	Nodes map[string]RPCNode
}

func LoadConfig(path string) (Config, error) {
	tree, err := toml.LoadFile(path)
	if err != nil {
		return Config{}, err
	}

	var config Config
	if err := tree.Unmarshal(&config); err != nil {
		return Config{}, err
	}

	return config, nil
}
