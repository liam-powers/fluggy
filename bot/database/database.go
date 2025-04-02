package database

import (
	"database/sql"
	"encoding/json"
	"fluggy-bot/types"
	"fluggy-bot/utils"
	"log"

	"github.com/lib/pq"
)

func getDbConn() *sql.DB {
	postgresUrl := utils.GetEnvVariable("POSTGRES_URL")
	db, err := sql.Open("postgres", postgresUrl)

	if err != nil {
		log.Fatalln("Err establishing DB conn: ", err)
	}
	if err = db.Ping(); err != nil {
		log.Fatalln("Error pinging database:", err)
	}

	return db
}

func userExists(discordId string) bool {
	db := getDbConn()

	rows, err := db.Query("SELECT * FROM users WHERE discord_id=$1;", discordId)
	if err != nil {
		log.Fatalln("Error querying database:", err)
	}

	if err := rows.Err(); err != nil {
		log.Panic("Error with rows retrieved:", err)
	}

	if rows.Next() {
		return true
	}

	return false
}

func AddOrUpdateUser(user types.User) {
	db := getDbConn()

	mutualServersPqArray := pq.Array(user.MutualServers)
	playerCardInfoJSON, err := json.Marshal(user.PlayerCardInfo)

	if userExists(user.DiscordId) {
		// UPDATE w/ new steam_id
		_, err := db.Exec("UPDATE users SET steam_id=$1, pfp_url=$2, mutual_servers=$3, elo=$4, global_rank_lb=$5, global_rank_ub=$6, timestamp=$7, wants_nickname_updates=$8, player_card_info=$9", user.SteamId, user.PfpUrl, mutualServersPqArray, user.Elo, user.GlobalRankLb, user.GlobalRankUb, user.Timestamp, user.WantsNicknameUpdates, playerCardInfoJSON)
		if err != nil {
			log.Fatalln("Error updating user entry in database", err)
		}
	} else {
		// INSERT
		if err != nil {
			log.Fatal("Error marshalling user.PlayerCardInfo into JSON", err)
		}
		_, err = db.Exec("INSERT INTO users (discord_id, steam_id, pfp_url, mutual_servers, elo, global_rank_lb, global_rank_ub, timestamp, wants_nickname_updates, player_card_info) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10);",
			user.DiscordId, user.SteamId, user.PfpUrl, mutualServersPqArray, user.Elo, user.GlobalRankLb, user.GlobalRankUb, user.Timestamp, user.WantsNicknameUpdates, playerCardInfoJSON)
		if err != nil {
			log.Fatalln("Error adding user to database", err)
		}
	}
}

func GetUsers() []types.User {
	db := getDbConn()
	rows, err := db.Query("SELECT * FROM users;")
	if err != nil {
		log.Fatalln("Error selecting users from database", err)
	}

	var users []types.User

	for rows.Next() {
		var user types.User
		var playerCardInfoJSON []byte
		var mutualServers pq.StringArray

		err := rows.Scan(
			&user.DiscordId,
			&user.SteamId,
			&user.PfpUrl,
			&mutualServers,
			&user.Elo,
			&user.GlobalRankLb,
			&user.GlobalRankUb,
			&user.Timestamp,
			&user.WantsNicknameUpdates,
			&playerCardInfoJSON,
		)

		if err != nil {
			log.Fatalln("Error scanning user row", err)
		}

		// Convert pq.StringArray to []string
		user.MutualServers = []string(mutualServers)

		// Parse the player card info JSON
		if err := json.Unmarshal(playerCardInfoJSON, &user.PlayerCardInfo); err != nil {
			log.Fatalln("Error unmarshalling player card info", err)
		}

		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		log.Fatalln("Error after scanning rows", err)
	}

	defer rows.Close()
	return users
}

func RemoveUser(discordId string) {
	if !userExists(discordId) { // no need to remove a non-existent user from database
		return
	}

	db := getDbConn()
	// remove user from database with DELETE
	_, err := db.Exec("DELETE FROM users WHERE discord_id=$1", discordId)

	if err != nil {
		log.Fatalln("Error removing user with discord_id", discordId, "from database", err)
	}
}
