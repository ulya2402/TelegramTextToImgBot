package app

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

type BotApp struct {
	BotToken  string
	DB        *Database
	Replicate *ReplicateConfig
	I18n      *I18nManager
	Providers []Provider
	Models    []ModelConfig
}

func NewBotApp(token string, db *Database, rep *ReplicateConfig, i18n *I18nManager) *BotApp {
	app := &BotApp{
		BotToken:  token,
		DB:        db,
		Replicate: rep,
		I18n:      i18n,
	}
	app.loadConfig()
	return app
}

func (b *BotApp) loadConfig() {
	pContent, err := os.ReadFile("config/Providers.json")
	if err != nil {
		log.Fatal("Failed to load Providers.json")
	}
	json.Unmarshal(pContent, &b.Providers)

	mContent, err := os.ReadFile("config/models.json")
	if err != nil {
		log.Fatal("Failed to load models.json")
	}
	json.Unmarshal(mContent, &b.Models)
	
	fmt.Printf("[INFO] Loaded %d providers and %d models.\n", len(b.Providers), len(b.Models))
}

func (b *BotApp) GetModelByID(id string) ModelConfig {
	for _, m := range b.Models {
		if m.ID == id {
			return m
		}
	}
	return ModelConfig{}
}