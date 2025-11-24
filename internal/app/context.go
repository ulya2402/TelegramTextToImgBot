package app

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings" // Import strings
)

type BotApp struct {
	BotToken    string
	SupabaseURL string
	SupabaseKey string
	DB          *Database
	Replicate   *ReplicateConfig
	I18n        *I18nManager
	Providers   []Provider
	Models      []ModelConfig
}

func NewBotApp(token, sbURL, sbKey string, db *Database, rep *ReplicateConfig, i18n *I18nManager) *BotApp {
	// SANITASI URL: Hapus slash di akhir jika ada
	cleanSbURL := strings.TrimRight(sbURL, "/")

	app := &BotApp{
		BotToken:    token,
		SupabaseURL: cleanSbURL, // Gunakan URL yang bersih
		SupabaseKey: sbKey,
		DB:          db,
		Replicate:   rep,
		I18n:        i18n,
	}
	
	app.loadConfig()
	
	if err := app.EnsureBucketExists(); err != nil {
		log.Fatalf("[FATAL] Gagal menginisialisasi Storage Bucket: %v", err)
	}

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