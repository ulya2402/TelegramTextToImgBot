package app

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/supabase-community/supabase-go"
)

type User struct {
	ID            int64                  `json:"id"`
	LanguageCode  string                 `json:"language_code"`
	Credits       int                    `json:"credits"`
	LastResetDate string                 `json:"last_reset_date"`
	CurrentState  string                 `json:"current_state"`
	SelectedModel string                 `json:"selected_model"`
	DraftConfig   map[string]interface{} `json:"draft_config"`
}

type Database struct {
	client *supabase.Client
}

func NewDatabase(url, key string) (*Database, error) {
	client, err := supabase.NewClient(url, key, nil)
	if err != nil {
		return nil, err
	}
	return &Database{client: client}, nil
}

func (db *Database) GetOrCreateUser(telegramID int64) (*User, error) {
	data, _, err := db.client.From("users").Select("*", "exact", false).Eq("id", fmt.Sprintf("%d", telegramID)).Execute()
	if err != nil {
		return nil, err
	}

	var users []User
	json.Unmarshal(data, &users)

	if len(users) == 0 {
		newUser := User{
			ID:            telegramID,
			LanguageCode:  "en",
			Credits:       5,
			LastResetDate: time.Now().UTC().Format("2006-01-02"),
			CurrentState:  "",
			DraftConfig:   make(map[string]interface{}),
		}
		_, _, err := db.client.From("users").Insert(newUser, false, "", "", "").Execute()
		if err != nil {
			return nil, err
		}
		return &newUser, nil
	}

	user := &users[0]
	if user.DraftConfig == nil {
		user.DraftConfig = make(map[string]interface{})
	}
	
	today := time.Now().UTC().Format("2006-01-02")
	if user.LastResetDate != today {
		updates := map[string]interface{}{
			"credits":         5,
			"last_reset_date": today,
		}
		db.client.From("users").Update(updates, "", "").Eq("id", fmt.Sprintf("%d", telegramID)).Execute()
		user.Credits = 5
	}

	return user, nil
}

func (db *Database) UpdateState(telegramID int64, state string, modelKey string) error {
	updates := map[string]interface{}{
		"current_state":  state,
		"selected_model": modelKey,
		"draft_config":   make(map[string]interface{}), // RESET DATA (Hati-hati)
	}
	_, _, err := db.client.From("users").Update(updates, "", "").Eq("id", fmt.Sprintf("%d", telegramID)).Execute()
	return err
}

// [FUNGSI BARU] UpdateCurrentState: Hanya ganti status, DRAFT TETAP AMAN (Dipakai navgasi menu)
func (db *Database) UpdateCurrentState(telegramID int64, state string) error {
	updates := map[string]interface{}{
		"current_state": state,
	}
	_, _, err := db.client.From("users").Update(updates, "", "").Eq("id", fmt.Sprintf("%d", telegramID)).Execute()
	return err
}

func (db *Database) UpdateDraftConfig(telegramID int64, key string, value interface{}) error {
	// Fetch current to merge
	user, err := db.GetOrCreateUser(telegramID)
	if err != nil {
		return err
	}
	
	user.DraftConfig[key] = value
	
	updates := map[string]interface{}{
		"draft_config": user.DraftConfig,
	}
	_, _, err = db.client.From("users").Update(updates, "", "").Eq("id", fmt.Sprintf("%d", telegramID)).Execute()
	return err
}

func (db *Database) ClearState(telegramID int64) error {
	updates := map[string]interface{}{
		"current_state":  "",
		"selected_model": "",
		"draft_config":   make(map[string]interface{}),
	}
	_, _, err := db.client.From("users").Update(updates, "", "").Eq("id", fmt.Sprintf("%d", telegramID)).Execute()
	return err
}

func (db *Database) DeductCredit(telegramID int64, amount int) error {
	user, err := db.GetOrCreateUser(telegramID)
	if err != nil {
		return err
	}
	if user.Credits < amount {
		return fmt.Errorf("insufficient_credits")
	}
	newCredits := user.Credits - amount
	_, _, err = db.client.From("users").Update(map[string]interface{}{"credits": newCredits}, "", "").Eq("id", fmt.Sprintf("%d", telegramID)).Execute()
	return err
}

func (db *Database) AddCredit(telegramID int64, amount int) error {
	user, err := db.GetOrCreateUser(telegramID)
	if err != nil {
		return err
	}
	_, _, err = db.client.From("users").Update(map[string]interface{}{"credits": user.Credits + amount}, "", "").Eq("id", fmt.Sprintf("%d", telegramID)).Execute()
	return err
}

func (db *Database) SetLanguage(telegramID int64, lang string) error {
	_, _, err := db.client.From("users").Update(map[string]interface{}{"language_code": lang}, "", "").Eq("id", fmt.Sprintf("%d", telegramID)).Execute()
	return err
}