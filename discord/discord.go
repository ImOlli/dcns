package discord

import (
	"dcns/model"
	"github.com/bwmarrin/discordgo"
	"log"
	"os"
	"os/exec"
	"os/signal"
)

var stop chan os.Signal
var config *model.Config
var ctxs = make(map[string]*model.DockerUpdateContext)

func StartDiscordBot(updateChannel chan *model.DockerUpdateContext, c *model.Config) {
	log.Println("Starting bot...")
	config = c
	discord, err := discordgo.New("Bot " + config.DiscordToken)

	if err != nil {
		panic(err)
	}

	discord.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Println("Bot is up!")
	})

	discord.AddHandler(handleInteraction)

	err = discord.Open()

	if err != nil {
		log.Fatalf("Cannot open the session: %v", err)
	}

	defer discord.Close()

	stop = make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	go waitForUpdate(updateChannel, discord)

	<-stop
	log.Println("Graceful shutdown")
}

func waitForUpdate(updateChannel chan *model.DockerUpdateContext, s *discordgo.Session) {
	for {
		select {
		case <-stop:
			return
		case context := <-updateChannel:
			createUpdateNotification(s, context)
		}
	}
}

func createUpdateNotification(s *discordgo.Session, context *model.DockerUpdateContext) {
	message, err := s.ChannelMessageSendComplex(config.DiscordChannelId, &discordgo.MessageSend{
		Content: "A new update is available!",
		Embed: &discordgo.MessageEmbed{
			Title: "A new version of " + context.ContainerContext.Image + " has been published",
			Author: &discordgo.MessageEmbedAuthor{
				Name: "DiscordUpdateNotificationService",
			},
			Footer: &discordgo.MessageEmbedFooter{
				Text: "powered by crazy-max/diun",
			},
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:  "Name",
					Value: context.ContainerContext.Name,
				},
				{
					Name:  "Image",
					Value: context.ContainerContext.Image,
				},
				{
					Name:  "Path",
					Value: context.ContainerContext.Path,
				},
				{
					Name:  "Created",
					Value: context.Created,
				},
				{
					Name:  "Digest",
					Value: context.Digest,
				},
				{
					Name:  "Hostname",
					Value: context.Hostname,
				},
				{
					Name:  "Docker Repository Link",
					Value: context.HubLink,
				},
			},
		},
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    "Apply update (Restart Container)",
						Style:    discordgo.SuccessButton,
						CustomID: "btn.confirm",
					},
					discordgo.Button{
						Label:    "Ignore",
						Style:    discordgo.SecondaryButton,
						CustomID: "btn.ignore",
					},
				},
			},
		},
	})

	if err != nil {
		log.Fatalln("Error while sending message ", err)
	}

	ctxs[message.ID] = context
}

func handleInteraction(s *discordgo.Session, r *discordgo.InteractionCreate) {
	if r.Type == discordgo.InteractionMessageComponent {
		c, exists := ctxs[r.Message.ID]

		if exists {
			err := removeButtons(s, r)

			if err != nil {
				//TODO Handle error
				log.Println("Error while editing previous integration message", err)
				return
			}

			if r.MessageComponentData().CustomID == "btn.confirm" {
				err = s.InteractionRespond(r.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
				})

				if err != nil {
					log.Println("Error while responding to interaction", err)
					_, _ = s.ChannelMessageSend(r.Message.ChannelID, "Failed to create discord interaction!")
				}

				// TODO Handle Update?
				err = restartContainerAndLog(c)

				if err != nil {
					log.Println("Error while restarting container", err)

					_, _ = s.FollowupMessageCreate(r.Interaction, false, &discordgo.WebhookParams{
						Content: "Failed to restart container :x:",
					})

					_ = addRetryButton(s, r)

					return
				}

				delete(ctxs, r.Message.ID)

				_, err = s.FollowupMessageCreate(r.Interaction, false, &discordgo.WebhookParams{
					Content: "Successfully restarted container! :white_check_mark:",
				})

				if err != nil {
					//TODO Handle error
					log.Println("Error while sending interaction response: ", err)
					return
				}
			} else if r.MessageComponentData().CustomID == "btn.ignore" {
				err = s.InteractionRespond(r.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Update ignored!",
					},
				})

				if err != nil {
					log.Println("Error while responding to interaction", err)
					_, _ = s.ChannelMessageSend(r.Message.ChannelID, "Failed to create discord interaction!")
				}

				delete(ctxs, r.Message.ID)
			}
		} else {
			err := s.InteractionRespond(r.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "The requested update does not exist anymore or is invalid",
				},
			})

			if err != nil {
				//TODO Handle error
				log.Println("Error while sending interaction response: ", err)
				return
			}
		}
	}
}

func addRetryButton(s *discordgo.Session, r *discordgo.InteractionCreate) error {
	_, err := s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Style:    discordgo.PrimaryButton,
						Label:    "Retry Update (Restart Container)",
						CustomID: "btn.confirm",
					},
					discordgo.Button{
						Label:    "Ignore",
						Style:    discordgo.SecondaryButton,
						CustomID: "btn.ignore",
					},
				},
			},
		},
		Content: &r.Message.Content,
		Embeds:  r.Message.Embeds,
		ID:      r.Message.ID,
		Channel: r.Message.ChannelID,
	})

	return err
}

func removeButtons(s *discordgo.Session, r *discordgo.InteractionCreate) error {
	_, err := s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Components: []discordgo.MessageComponent{},
		Content:    &r.Message.Content,
		Embeds:     r.Message.Embeds,
		ID:         r.Message.ID,
		Channel:    r.Message.ChannelID,
	})

	return err
}

func restartContainerAndLog(c *model.DockerUpdateContext) error {
	log.Println("Restarting Container " + c.ContainerContext.Image + " on path " + c.ContainerContext.Path)

	cmd := exec.Command("docker", "compose", "down")
	cmd.Dir = c.ContainerContext.Path
	stdout, err := cmd.CombinedOutput()
	log.Println(string(stdout))

	if err != nil {
		return err
	}

	cmd = exec.Command("docker", "compose", "pull")
	cmd.Dir = c.ContainerContext.Path
	stdout, err = cmd.CombinedOutput()
	log.Println(string(stdout))

	if err != nil {
		return err
	}

	cmd = exec.Command("docker", "compose", "pull")
	cmd.Dir = c.ContainerContext.Path
	stdout, err = cmd.CombinedOutput()
	log.Println(string(stdout))

	log.Println("Done!")

	return err
}
