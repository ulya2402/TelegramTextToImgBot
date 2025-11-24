package app

import (
	"fmt"
	"strings"
	"time"
)

func (b *BotApp) HandleMessage(update TelegramUpdate) {
	userID := update.Message.From.ID
	chatID := update.Message.Chat.ID
	text := update.Message.Text

	user, err := b.DB.GetOrCreateUser(userID)
	if err != nil { return }

	if len(update.Message.Photo) > 0 && user.CurrentState == "waiting_prompt" {
		photoID := update.Message.Photo[len(update.Message.Photo)-1].FileID
		go b.DB.UpdateDraftConfig(userID, "image_input", photoID) 
		SendMessage(b.BotToken, chatID, b.I18n.Get(user.LanguageCode, "img_received"), nil)
		return
	}

	if strings.HasPrefix(text, "/") {
		if text == "/start" {
			go b.DB.ClearState(userID)
			SendMessage(b.BotToken, chatID, b.I18n.Get(user.LanguageCode, "welcome", user.Credits), nil)
			return
		}
		if text == "/img" {
			go b.DB.ClearState(userID)
			b.ShowProviderList(chatID, user, false, 0)
			return
		}
		if text == "/profile" || text == "/status" {
			now := time.Now().UTC()
			tomorrow := now.Add(24 * time.Hour)
			nextReset := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 0, 0, 0, 0, time.UTC)
			duration := nextReset.Sub(now)
			hours := int(duration.Hours())
			minutes := int(duration.Minutes()) % 60
			timeString := fmt.Sprintf("%dh %dm", hours, minutes)

			msg := b.I18n.Get(user.LanguageCode, "profile_msg", user.ID, user.Credits, timeString)
			SendMessage(b.BotToken, chatID, msg, nil)
			return
		}
	}

	if user.CurrentState == "waiting_prompt" && user.SelectedModel != "" {
		b.ProcessImageGeneration(user, chatID, text)
		return
	}

	SendMessage(b.BotToken, chatID, b.I18n.Get(user.LanguageCode, "use_img_cmd"), nil)
}

func (b *BotApp) HandleCallback(update TelegramUpdate) {
	data := update.CallbackQuery.Data
	userID := update.CallbackQuery.From.ID
	chatID := update.CallbackQuery.Message.Chat.ID
	msgID := update.CallbackQuery.Message.MessageID

	AnswerCallback(b.BotToken, update.CallbackQuery.ID)
	
	if strings.HasPrefix(data, "lang_") {
		lang := strings.TrimPrefix(data, "lang_")
		b.DB.SetLanguage(userID, lang)
		SendMessage(b.BotToken, chatID, "Language updated.", nil)
		return
	}

	user, _ := b.DB.GetOrCreateUser(userID)

	if data == "nav_providers" || data == "nav_cancel" {
		go b.DB.ClearState(userID)
		b.ShowProviderList(chatID, user, true, msgID)
		return
	}

	if strings.HasPrefix(data, "prov_") {
		provID := strings.TrimPrefix(data, "prov_")
		var buttons []map[string]string
		for _, m := range b.Models {
			if !m.Enabled { continue }
			isMatch := false
			if strings.HasPrefix(m.ReplicateID, "google/") && provID == "google" {
				isMatch = true
			} else {
				parts := strings.Split(m.ReplicateID, "/")
				if len(parts) > 0 && parts[0] == provID {
					isMatch = true
				}
			}

			if isMatch {
				buttons = append(buttons, map[string]string{
					"text": fmt.Sprintf("%s (%d Cr)", m.Name, m.Cost), 
					"callback_data": "model_" + m.ID,
				})
			}
		}
		buttons = append(buttons, map[string]string{
			"text": b.I18n.Get(user.LanguageCode, "back_to_prov"),
			"callback_data": "nav_providers",
		})
		
		msg := b.I18n.Get(user.LanguageCode, "select_model")
		if len(buttons) == 1 {
			msg = b.I18n.Get(user.LanguageCode, "model_unavailable")
		}
		EditMessageText(b.BotToken, chatID, msgID, msg, buttons)
		return
	}

	if strings.HasPrefix(data, "model_") {
		modelID := strings.TrimPrefix(data, "model_")
		modelConf := b.GetModelByID(modelID)
		if modelConf.ID == "" { return }

		go func() {
			b.DB.UpdateState(userID, "waiting_prompt", modelID)
			for _, p := range modelConf.Parameters {
				if p.Default != nil {
					b.DB.UpdateDraftConfig(userID, p.Name, p.Default)
				}
			}
		}()
		
		user.DraftConfig = make(map[string]interface{})
		for _, p := range modelConf.Parameters {
			if p.Default != nil {
				user.DraftConfig[p.Name] = p.Default
			}
		}
		b.ShowModelPanel(chatID, msgID, user, modelConf)
		return
	}

	if strings.HasPrefix(data, "set_open|") {
		paramName := strings.TrimPrefix(data, "set_open|")
		modelConf := b.GetModelByID(user.SelectedModel)
		b.ShowSettingOptions(chatID, msgID, user, paramName, modelConf)
		return
	}

	if strings.HasPrefix(data, "set_val|") {
		content := strings.TrimPrefix(data, "set_val|")
		parts := strings.SplitN(content, "|", 2)
		if len(parts) == 2 {
			paramName := parts[0]
			val := parts[1]
			go b.DB.UpdateDraftConfig(userID, paramName, val)
			user.DraftConfig[paramName] = val
			modelConf := b.GetModelByID(user.SelectedModel)
			b.ShowModelPanel(chatID, msgID, user, modelConf)
		}
		return
	}

	if data == "back_to_panel" {
		modelConf := b.GetModelByID(user.SelectedModel)
		b.ShowModelPanel(chatID, msgID, user, modelConf)
		return
	}
}