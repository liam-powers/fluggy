package main

import (
	"encoding/json"
	"fluggy-bot/cron"
	"fluggy-bot/database"
	"fluggy-bot/types"
	"fluggy-bot/utils"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

type optionMap = map[string]*discordgo.ApplicationCommandInteractionDataOption

func parseOptions(options []*discordgo.ApplicationCommandInteractionDataOption) (om optionMap) {
	om = make(optionMap)
	for _, opt := range options {
		om[opt.Name] = opt
	}
	return
}

func interactionAuthor(i *discordgo.Interaction) *discordgo.User {
	if i.Member != nil {
		return i.Member.User
	}
	return i.User
}

func handleEcho(s *discordgo.Session, i *discordgo.InteractionCreate, opts optionMap) {
	builder := new(strings.Builder)
	if v, ok := opts["author"]; ok && v.BoolValue() {
		author := interactionAuthor(i.Interaction)
		builder.WriteString("**" + author.String() + "** says: ")
	}
	builder.WriteString(opts["message"].StringValue())
	log.Println("Builder string: ", builder.String())

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: builder.String(),
		},
	})

	if err != nil {
		log.Panicf("could not respond to interaction: %s", err)
	}
}

func handleAddSteam(s *discordgo.Session, i *discordgo.InteractionCreate, opts optionMap) {
	utils.SendProcessingMessage(s, i)

	steamUrl := opts["steam_url"].StringValue()
	wantsNicknameUpdates := opts["wants_nickname_updates"].BoolValue()
	primaryColor := opts["primary_color"].StringValue()
	secondaryColor := opts["secondary_color"].StringValue()
	author := interactionAuthor(i.Interaction)

	// Resolve vanity URL's via Steam API
	// example: https://steamcommunity.com/id/liamhi/
	var steamSuffix string
	steamUrlParts := strings.Split(steamUrl, "/")
	if steamUrlParts[len(steamUrlParts)-1] == "" {
		steamSuffix = steamUrlParts[len(steamUrlParts)-2]
	} else {
		steamSuffix = steamUrlParts[len(steamUrlParts)-1]
	}

	resolveVanityURL := fmt.Sprintf("https://api.steampowered.com/ISteamUser/ResolveVanityURL/v0001/?key=%s&vanityurl=%s", SteamAPIKey, steamSuffix)
	res, err := http.Get(resolveVanityURL)
	if err != nil {
		log.Printf("Error making steam API request for author %s", author.String())
		return
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		log.Printf("Bad status code for author %s: %d", author, res.StatusCode)
		return
	}

	var response struct {
		Response struct {
			Success int    `json:"success"`
			SteamID string `json:"steamid,omitempty"`
			Message string `json:"message,omitempty"`
		} `json:"response"`
	}

	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		log.Printf("Error parsing response: %v", err)
		return
	}

	if response.Response.Success != 1 {
		log.Printf("Couldn't resolve vanity URL for author %s", author.String())
		return
	}

	steamId := response.Response.SteamID
	pfpUrl := i.Member.User.AvatarURL("")
	mutualServers := utils.GetMutualServers(s, author.ID)
	timestamp := time.Now().Unix()
	playerCardInfo := make(map[string]string)
	playerCardInfo["primary_color"] = primaryColor
	playerCardInfo["secondary_color"] = secondaryColor

	newUser := types.User{
		DiscordId:            author.ID,
		SteamId:              steamId,
		PfpUrl:               pfpUrl,
		MutualServers:        mutualServers,
		Elo:                  -1,
		GlobalRankLb:         0,
		GlobalRankUb:         0,
		Timestamp:            int(timestamp),
		WantsNicknameUpdates: wantsNicknameUpdates,
		PlayerCardInfo:       playerCardInfo,
	}

	database.AddOrUpdateUser(newUser)
	cron.UpdatePlayerDataCron(s)
	utils.EditToDoneMessage(s, i)
}

func handleRemoveSteam(s *discordgo.Session, i *discordgo.InteractionCreate) {
	utils.SendProcessingMessage(s, i)

	author := interactionAuthor(i.Interaction)
	database.RemoveUser(author.ID)

	utils.EditToDoneMessage(s, i)
}

var colorChoices = []*discordgo.ApplicationCommandOptionChoice{
	{
		Name:  "Red",
		Value: "red",
	},
	{
		Name:  "Green",
		Value: "green",
	},
	{
		Name:  "Blue",
		Value: "blue",
	},
	{
		Name:  "Yellow",
		Value: "yellow",
	},
	{
		Name:  "Cyan",
		Value: "cyan",
	},
	{
		Name:  "Pink",
		Value: "pink",
	},
	{
		Name:  "Orange",
		Value: "orange",
	},
	{
		Name:  "White",
		Value: "white",
	},
	{
		Name:  "Black",
		Value: "black",
	},
	{
		Name:  "Grey",
		Value: "grey",
	},
	{
		Name:  "Beige",
		Value: "beige",
	},
}

var commands = []*discordgo.ApplicationCommand{
	{
		Name:        "echo",
		Description: "Say something through a bot",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "message",
				Description: "Contents of the message",
				Type:        discordgo.ApplicationCommandOptionString,
				Required:    true,
			},
			{
				Name:        "author",
				Description: "Whether to prepend message's author",
				Type:        discordgo.ApplicationCommandOptionBoolean,
			},
		},
	},
	{
		Name:        "add_steam",
		Description: "Get on the leaderboard by sending your Steam URL",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "steam_url",
				Description: "Your Steam URL (can be found on Steam client above your profile page)",
				Type:        discordgo.ApplicationCommandOptionString,
				Required:    true,
			},
			{
				Name:        "wants_nickname_updates",
				Description: "If you'd like your current ELO to be appended to your Discord nickname",
				Type:        discordgo.ApplicationCommandOptionBoolean,
				Required:    true,
			},
			{
				Name:        "primary_color",
				Description: "The primary color used for your leaderboard display entry",
				Type:        discordgo.ApplicationCommandOptionString,
				Required:    true,
				Choices:     colorChoices,
			},
			{
				Name:        "secondary_color",
				Description: "The secondary color used for your leaderboard display entry",
				Type:        discordgo.ApplicationCommandOptionString,
				Required:    true,
				Choices:     colorChoices,
			},
		},
	},
	{
		Name:        "remove_steam",
		Description: "Remove yourself from the leaderboard",
	},
}

var (
	Token       = utils.GetEnvVariable("PRIVATE_KEY")
	App         = utils.GetEnvVariable("APP_ID")
	Guild       = utils.GetEnvVariable("TESTING_GUILD")
	SteamAPIKey = utils.GetEnvVariable("STEAM_API_KEY")
)

func main() {
	session, _ := discordgo.New("Bot " + Token)

	session.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if i.Type != discordgo.InteractionApplicationCommand {
			return
		}

		data := i.ApplicationCommandData()
		switch data.Name {
		case "echo":
			handleEcho(s, i, parseOptions(data.Options))
		case "add_steam":
			handleAddSteam(s, i, parseOptions(data.Options))
		case "remove_steam":
			handleRemoveSteam(s, i)
		default:
			return
		}
	})

	session.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("Logged in as %s", r.User.String())

		// run once immediately when bot starts
		cron.UpdatePlayerDataCron(s)

		// ticker every 5 minutes
		ticker := time.NewTicker(5 * time.Minute)
		go func() {
			for range ticker.C {
				cron.UpdatePlayerDataCron(s)
			}
		}()
	})

	_, err := session.ApplicationCommandBulkOverwrite(App, Guild, commands)
	if err != nil {
		log.Fatalf("could not register commands: %s", err)
	}

	err = session.Open()
	if err != nil {
		log.Fatalf("could not open session: %s", err)
	}

	sigch := make(chan os.Signal, 1)
	signal.Notify(sigch, os.Interrupt)
	<-sigch

	err = session.Close()
	if err != nil {
		log.Printf("could not close session gracefully: %s", err)
	}
}
