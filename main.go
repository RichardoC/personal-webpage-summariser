package main

import (
	"context"
	"errors"
	"fmt"
	"io"
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

	// doc.Find("img").Remove()

	doc.Find("img").Each(func(i int, el *goquery.Selection) {
		slog.Error(fmt.Sprintf("Found img %+v", el))
		fmt.Println()
		fmt.Println()
		fmt.Println()
		el.Remove()
	})
	// We don't care about images, we only want the text on the site
	doc.Find("picture").Each(func(i int, el *goquery.Selection) {
		el.Remove()
	})
	// We don't care about scripts, we only want the text on the site
	doc.Find("script").Each(func(i int, el *goquery.Selection) {
		el.Remove()
	})
	// We don't care about css, we only want the text on the site
	doc.Find("style").Each(func(i int, el *goquery.Selection) {
		el.Remove()
	})
	// We don't care about videos, we only want the text on the site
	doc.Find("video").Each(func(i int, el *goquery.Selection) {
		el.Remove()
	})

	// remove excess newlines etc
	bdyText := standardizeSpaces(doc.Text())
	slog.Info(bdyText)
	return bdyText, err
}

func doSummarisation(token string, maxTokens int, server string, model string, siteText string) (summary string, err error) {

	lengthOfDataToSend := len(siteText)

	// Just a rule of thumb
	if len(siteText) > maxTokens {
		lengthOfDataToSend = 2 * maxTokens
	}

	aiClient := openai.DefaultConfig(token)
	aiClient.BaseURL = server
	client := openai.NewClientWithConfig(aiClient)

	stream, err := client.CreateChatCompletionStream(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: model,
			Messages: []openai.ChatCompletionMessage{
				{Role: openai.ChatMessageRoleSystem,
					Content: "You are an expert at summarising text in a concise manner for intelligent social media users.",
				},
				{
					Role: openai.ChatMessageRoleUser,
					Content: fmt.Sprintf("Summarise the following text using the fewest possible words\n, %s",
						// siteText),
						siteText[:lengthOfDataToSend]),
				},
			},
			Stream:           true,
			Stop:             []string{"<|end_of_text|>", "<|eot_id|>"},
			Temperature:      0.7,
			TopP:             0.95,
			MaxTokens:        maxTokens,
			FrequencyPenalty: 0.0,
			PresencePenalty:  0.0,
		},
	)

	if err != nil {
		slog.Error("Error with ChatCompletionStream for summariser", "error", err)
		return "", err
	}
	defer stream.Close()

	var summaryBuilder strings.Builder
	for {
		var response openai.ChatCompletionStreamResponse
		response, err = stream.Recv()
		if errors.Is(err, io.EOF) {
			slog.Debug("Finished receiving response from LLM")
			return summaryBuilder.String(), nil
		}

		if err != nil {
			slog.Error("Error with ChatCompletionStream for summariser", "error", err)
			return summaryBuilder.String(), err
		}
		_, err = summaryBuilder.WriteString(response.Choices[0].Delta.Content)

		if err != nil {
			slog.Error("Error with writing stream to summary", "error", err)
			return summaryBuilder.String(), err
		}

	}

}

func main() {

	// Replace this with a function to discover
	// site := "https://www.bbc.co.uk/news/uk-england-london-68552817"
	site := "https://github.com/PuerkitoBio/goquery"
	// site := "https://example.com"

	siteText, err := scrapeSite(site)

	// error already logged
	if err != nil {
		os.Exit(4)
	}

	slog.Debug("site text obtained", "text", siteText)

	token := "fake-token"
	server := "http://127.0.0.1:1337/v1"
	model := "llama3-8b-instruct"
	maxTokens := 4096

	summary, err := doSummarisation(token, maxTokens, server, model, siteText)
	// error already logged
	if err != nil {
		os.Exit(4)
	}

	slog.Debug("summary text obtained", "text", summary)

	fmt.Println(summary)
	reduction := float64(len(siteText) - len(summary))
	lengthRatio := (reduction) / float64(len(siteText))

	fmt.Printf("The original site contains %d words, the summary contains %d words. Saved %f %%\n", len(siteText), len(summary), lengthRatio*100)
}
