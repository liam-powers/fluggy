package utils

import (
	"log"
	"os"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

func GetEnvVariable(name string) string {
	err := godotenv.Load()
	if err != nil {
		log.Fatalln("Error loading .env file")
	}

	return os.Getenv(name)
}

func SendProcessingMessage(s *discordgo.Session, i *discordgo.InteractionCreate) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Processing...",
		},
	})
	if err != nil {
		log.Fatalln("Error sending interaction: ", err)
	}
}

func EditToDoneMessage(s *discordgo.Session, i *discordgo.InteractionCreate) {
	responseContent := "Done."
	_, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &responseContent,
	})
	if err != nil {
		log.Fatalln("Error updating interaction:", err)
	}
}

func GetMutualServers(s *discordgo.Session, userId string) []string {
	mutualServers := []string{}

	// all guilds BOT is in
	guilds, err := s.UserGuilds(100, "", "", false) // NOTE / TODO: fetches up to 100 guilds
	if err != nil {
		log.Printf("Error fetching bot guilds: %v", err)
	}

	for _, guild := range guilds {
		_, err := s.GuildMember(guild.ID, userId)
		if err == nil {
			// no error means the user is a member of this guild
			mutualServers = append(mutualServers, guild.ID)
		}
	}

	return mutualServers
}
