package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	goopenai "github.com/CasualCodersProjects/gopenai"
	"github.com/CasualCodersProjects/gopenai/types"
	"github.com/joho/godotenv"
	tele "gopkg.in/telebot.v3"
)

func main() {
	var (
		// Universal markup builders.
		menu     = &tele.ReplyMarkup{ResizeKeyboard: true}
		selector = &tele.ReplyMarkup{}

		// Reply buttons.
		btnHelp     = menu.Text("ℹ Help")
		btnSettings = menu.Text("⚙ Settings")

		// Inline buttons.
		//
		// Pressing it will cause the client to
		// send the bot a callback.
		//
		// Make sure Unique stays unique as per button kind
		// since it's required for callback routing to work.
		//
		btnPrev = selector.Data("⬅", "prev")
		btnNext = selector.Data("➡", "next")
	)

	godotenv.Load()
	pref := tele.Settings{
		Token:  os.Getenv("TELEGRAM_KEY"),
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		log.Fatal(err)
		return
	}

	menu.Reply(
		menu.Row(btnHelp),
		menu.Row(btnSettings),
	)

	selector.Inline(
		selector.Row(btnPrev, btnNext),
	)

	b.Handle("/start", func(c tele.Context) error {
		return c.Send("Hello!", menu)
	})

	// On reply button pressed (message)
	b.Handle(&btnHelp, func(c tele.Context) error {
		return c.Edit("Here is some help: ...")
	})

	// On inline button pressed (callback)
	b.Handle(&btnPrev, func(c tele.Context) error {
		return c.Respond()
	})

	b.Handle("/client", func(c tele.Context) error {
		fmt.Println(c.Message().Payload)

		resp, err := client(c.Message().Payload)
		if err != nil {
			return err
		}
		return c.Send(resp)
	})

	b.Handle(tele.OnText, func(c tele.Context) error {
		msg := c.Text
		resp, err := rsp(msg())
		if err != nil {
			return err
		}
		return c.Send(resp)
	})

	fmt.Println("bot start running")
	b.Start()
}

func rsp(question string) (response string, err error) {
	godotenv.Load(".env")
	openAI := goopenai.NewOpenAI(&goopenai.OpenAIOpts{
		APIKey: os.Getenv("AI_KEY"),
	})

	request := types.NewDefaultCompletionRequest("The following is a conversation with an AI assistant. The assistant is helpful, creative, clever, and very friendly.\n\nHuman: Hello, who are you?\nAI: I am an AI created by OpenAI. How can I help you today?\nHuman: " + question + "\nAI:")
	request.Model = "text-davinci-003"
	request.Temperature = 0.9
	request.MaxTokens = 150
	request.TopP = 1
	request.FrequencyPenalty = 0
	request.PresencePenalty = 0.6
	request.Stop = []string{" Human:", " AI:"}

	resp, err := openAI.CreateCompletion(request)
	if err != nil {
		return "", err
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("Response not Found!")
	}

	return resp.Choices[0].Text, nil
}

type JsonResponse struct {
	Ai     string `json:"ai"`
	Object string `json:"object"`
}

func client(message string) (resp string, err error) {
	req, err := http.NewRequest("GET", "http://localhost:8080/api/v1/ai/chat?human="+message, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	res, err := client.Do(req)

	if err != nil {
		return "", err
	}

	defer res.Body.Close()

	b, err := ioutil.ReadAll(res.Body)

	var jsonResponse []JsonResponse

	err = json.Unmarshal(b, &jsonResponse)
	if err != nil {
		return "", err
	}

	return jsonResponse[0].Ai, nil
}
