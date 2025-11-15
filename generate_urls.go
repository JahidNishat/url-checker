package main

import (
	"fmt"
	"os"
)

func main() {
	count := 10000 // Default 10k URLs
	if len(os.Args) > 1 {
		fmt.Sscanf(os.Args[1], "%d", &count)
	}

	file, _ := os.Create("urls.txt")
	defer file.Close()

	realDomains := []string{
		"google.com", "github.com", "stackoverflow.com",
		"reddit.com", "wikipedia.org", "youtube.com",
		"amazon.com", "twitter.com", "facebook.com",
		"linkedin.com", "microsoft.com", "apple.com",
		"netflix.com", "instagram.com", "tiktok.com",
		"twitch.tv", "discord.com", "slack.com",
	}

	for i := 0; i < count; i++ {
		domain := realDomains[i%len(realDomains)]
		fmt.Fprintf(file, "https://www.%s\n", domain)
	}

	fmt.Printf("✅ Generated %d URLs in urls.txt\n", count)
}
