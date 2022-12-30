package main

import (
	"dcns/discord"
	"dcns/model"
	"dcns/server"
)

func main() {
	updateChannel := make(chan *model.DockerUpdateContext)

	config := model.LoadConfig()

	go server.StartServer(updateChannel, &config)
	discord.StartDiscordBot(updateChannel, &config)
}
