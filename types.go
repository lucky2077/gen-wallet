package main

type Config struct {
	Prefix             string
	Suffix             string
	Concurrent         int
	LOG_LEVEL          int
	CountPerGeneration int
	DiscordWebhook     string
	RSAPublicKey       string
}

type Message struct {
	Content string `json:"content"`
}
