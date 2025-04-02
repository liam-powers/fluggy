package types

type User struct {
	DiscordId            string
	SteamId              string
	PfpUrl               string
	MutualServers        []string
	Elo                  int
	GlobalRankLb         int
	GlobalRankUb         int
	Timestamp            int
	WantsNicknameUpdates bool
	PlayerCardInfo       map[string]string
}

type Leaderboard struct {
	NextRequestURL string `xml:"nextRequestURL"`
	Entries        struct {
		Entry []struct {
			SteamID string `xml:"steamid"`
			Score   int    `xml:"score"`
			Rank    int    `xml:"rank"`
		} `xml:"entry"`
	} `xml:"entries"`
}
