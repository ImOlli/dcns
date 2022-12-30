package model

import (
	"gopkg.in/yaml.v3"
	"io"
	"os"
)

type Config struct {
	DiscordToken      string                    `yaml:"discordToken"`
	DiscordChannelId  string                    `yaml:"discordChannelId"`
	ContainerContexts *[]DockerContainerContext `yaml:"container"`
	Host              string                    `yaml:"host"`
	Port              string                    `yaml:"port"`
}

func LoadConfig() Config {
	jsonFile, err := os.Open("config.yaml")
	defer jsonFile.Close()
	// if we os.Open returns an error then handle it
	if err != nil {
		panic(err)
	}

	bytes, err := io.ReadAll(jsonFile)

	if err != nil {
		panic(err)
	}

	var config Config

	err = yaml.Unmarshal(bytes, &config)

	if err != nil {
		panic(err)
	}

	return config
}
