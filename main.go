package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/PuerkitoBio/goquery"
	openai "github.com/sashabaranov/go-openai"
)

func standardizeSpaces(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func scrapeSite(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		slog.Error("Failed to get site", "site", url, "error", err)
		return "", err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)

	if err != nil {
		slog.Error("Failed to get site body", "site", url, "error", err)
		return "", err
	}

	// We don't care about scripts, we only want the text on the site
	doc.Find("script").Each(func(i int, el *goquery.Selection) {
		el.Remove()
	})
	// We don't care about css, we only want the text on the site
	doc.Find("style").Each(func(i int, el *goquery.Selection) {
		el.Remove()
	})

	// remove excess newlines etc
	bdyText := standardizeSpaces(doc.Text())
	return bdyText, err
}

func doSummarisation(token string, server string, model string, siteText string) (summary string, err error){
	aiClient := openai.DefaultConfig(token)
	aiClient.BaseURL = server
	client := openai.NewClientWithConfig(aiClient)

	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: model,
			Messages: []openai.ChatCompletionMessage{
				{Role: openai.ChatMessageRoleSystem,
					Content: "You are an expert at summarising text in a concise manner for intelligent social media users.",
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: fmt.Sprintf("Summarise the following text\n, %s", siteText),
				},
			},
			Stop:             []string{"<|end_of_text|>", "<|eot_id|>"},
			Temperature:      0.7,
			TopP:             0.95,
			MaxTokens:        4096,
			FrequencyPenalty: 0.0,
			PresencePenalty:  0.0,
		},
	)
	if err != nil {
		slog.Error("Error with summarisation", "error", err)
		return "", err
	}

	return resp.Choices[0].Message.Content, nil

}

func main() {
	site := "http://example.com/"

	siteText, err := scrapeSite(site)

	if err != nil {
		os.Exit(4)
	}

	slog.Debug("site text obtained", "text", siteText)

	token := "fake-token"
	server := "http://127.0.0.1:1337/v1"
	model := "llama3-8b-instruct"

	fmt.Println(doSummarisation(token, server, model, siteText))
}
