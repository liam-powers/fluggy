package cron

import (
	"encoding/xml"
	"fluggy-bot/database"
	"fluggy-bot/types"
	"fluggy-bot/utils"
	"log"
	"net/http"

	"github.com/bwmarrin/discordgo"
)

func UpdatePlayerDataCron(s *discordgo.Session) {
	log.Println("cronning...")
	// 1. query database for a list of users
	users := database.GetUsers()

	// 2. create steamId to user map for O(1) lookups
	steamIdToUser := make(map[string]types.User)

	for _, user := range users {
		// NOTE / TODO: Some slowdown may happen here!
		user.MutualServers = utils.GetMutualServers(s, user.DiscordId) // making sure to update mutual servers for each user in case they join / leave ones with bot in there
		steamIdToUser[user.SteamId] = user
	}

	// 3. for each leaderboard entry, check if their steamId is in map, and if so,
	// 		modify user entry in map (also allows for O(1) access)
	// Fetch leaderboard entries from Steam API
	leaderboardPage := getLeaderboard("https://steamcommunity.com/stats/2217000/leaderboards/14800950/?xml=1")

	updates := 0

	// tracking lower bound and upper bound ranks:
	// - know previous elo
	// - maintain lowest rank with elo
	// - maintain highest rank with elo
	// - if curr elo == previous elo, add steam id to array and reassign highest_rank_with_elo to current rank
	// - once curr elo < previous elo, pop steam id's from array and index into dictionary
	// 		to give them all the lowest and highest rank with elo. then, reassign lowest and highest rank with elo to current rank.

	lowestRankWithElo := 1
	highestRankWithElo := 1
	steamIdsWithElo := []string{}
	var prevElo int
	isFirstEntry := true

	// continuously fetch leaderboard entries while nextRequestURL exists and there are still necessary users to parse
	for leaderboardPage.NextRequestURL != "" && (updates < len(steamIdToUser) && len(steamIdsWithElo) == 0) {
		nextPage := getLeaderboard(leaderboardPage.NextRequestURL)
		for _, entry := range leaderboardPage.Entries.Entry {
			if isFirstEntry {
				prevElo = entry.Score
				isFirstEntry = false
			}

			currElo := entry.Score
			if currElo == prevElo {
				highestRankWithElo = entry.Rank
			} else {
				// assign elo's for each user
				for _, steamId := range steamIdsWithElo {
					if trackedUser, exists := steamIdToUser[steamId]; exists {
						trackedUser.GlobalRankLb = lowestRankWithElo
						trackedUser.GlobalRankUb = highestRankWithElo
						steamIdToUser[steamId] = trackedUser
					}
				}

				// reset elo bounds and tracking
				steamIdsWithElo = []string{}
				lowestRankWithElo = entry.Rank
				highestRankWithElo = entry.Rank
			}

			if trackedUser, exists := steamIdToUser[entry.SteamID]; exists {
				steamIdsWithElo = append(steamIdsWithElo, trackedUser.SteamId)
				updates++
				// trackedUser.Rank = entry.Rank
				trackedUser.Elo = entry.Score
				steamIdToUser[entry.SteamID] = trackedUser
			}

			prevElo = currElo
		}

		// Update nextRequestURL for next iteration
		leaderboardPage.NextRequestURL = nextPage.NextRequestURL
	}

	// 4. then, AddOrUpdateUser() on each user in map
	for _, user := range steamIdToUser {
		database.AddOrUpdateUser(user)
	}
}

func getLeaderboard(requestURL string) types.Leaderboard {
	resp, err := http.Get(requestURL)
	if err != nil {
		log.Fatalln("Error fetching leaderboard data", err)
	}
	defer resp.Body.Close()

	// Parse XML response
	var leaderboard types.Leaderboard

	decoder := xml.NewDecoder(resp.Body)
	if err := decoder.Decode(&leaderboard); err != nil {
		log.Fatalln("Error parsing leaderboard XML", err)
	}

	return leaderboard
}
