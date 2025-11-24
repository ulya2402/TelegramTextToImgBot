package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	// Pastikan ini sesuai nama module di go.mod Anda
	"replicateReqBot/internal/app"

	"github.com/joho/godotenv"
)

func main() {
	fmt.Println("[INFO] Starting TelegramTextToImgBot v3.2 (Auto Bucket)...")
	godotenv.Load()

	// 1. Load Variables
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	sbURL := os.Getenv("SUPABASE_URL") // <-- Load URL Supabase
	sbKey := os.Getenv("SUPABASE_KEY") // <-- Load Key Supabase

	// Validasi agar tidak panic di tengah jalan
	if token == "" {
		log.Fatal("[FATAL] TELEGRAM_BOT_TOKEN is missing in .env")
	}
	if sbURL == "" || sbKey == "" {
		log.Fatal("[FATAL] SUPABASE_URL or SUPABASE_KEY is missing in .env")
	}

	// 2. Init Dependencies
	db, err := app.NewDatabase(sbURL, sbKey)
	if err != nil {
		log.Fatal("[FATAL] Database connection failed:", err)
	}

	i18n := app.NewI18nManager()
	i18n.LoadTranslations("locales")

	replicate := app.NewReplicate(os.Getenv("REPLICATE_API_TOKEN"))

	// 3. Init Bot App (The "Brain")
	// PERBAIKAN DISINI: Kita masukkan sbURL dan sbKey ke constructor
	bot := app.NewBotApp(token, sbURL, sbKey, db, replicate, i18n)

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

		if !tgResp.Ok {
			continue
		}

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