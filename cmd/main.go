package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"replicateReqBot/internal/app"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	fmt.Println("[INFO] Starting Business Bot v3.0 (Modular Architecture)...")
	godotenv.Load()

	// 1. Load Vars
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" { log.Fatal("TELEGRAM_BOT_TOKEN missing") }

	// 2. Init Dependencies
	db, err := app.NewDatabase(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_KEY"))
	if err != nil { log.Fatal(err) }

	i18n := app.NewI18nManager()
	i18n.LoadTranslations("locales")

	replicate := app.NewReplicate(os.Getenv("REPLICATE_API_TOKEN"))

	// 3. Init Bot App (The "Brain")
	bot := app.NewBotApp(token, db, replicate, i18n)

	// 4. Start Polling
	startPolling(bot)
}

func startPolling(bot *app.BotApp) {
	client := &http.Client{}
	url := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates", bot.BotToken)
	lastUpdateID := 0

	fmt.Println("[INFO] Bot is running. Waiting for messages...")

	for {
		reqData := map[string]interface{}{"offset": lastUpdateID + 1, "timeout": 60}
		jsonData, _ := json.Marshal(reqData)
		resp, err := client.Post(url, "application/json", bytes.NewBuffer(jsonData))
		
		if err != nil {
			fmt.Println("[ERROR] Polling:", err)
			time.Sleep(5 * time.Second)
			continue
		}

		var tgResp app.TelegramResponse
		json.NewDecoder(resp.Body).Decode(&tgResp)
		resp.Body.Close()

		if !tgResp.Ok { continue }

		for _, update := range tgResp.Result {
			lastUpdateID = update.UpdateID
			
			// Route to Handler Methods
			if update.CallbackQuery.ID != "" {
				go bot.HandleCallback(update) // Run in Goroutine for speed
			} else if update.Message.Text != "" || len(update.Message.Photo) > 0 {
				go bot.HandleMessage(update) // Run in Goroutine for speed
			}
		}
	}
}