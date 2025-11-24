package app

import (
	"fmt"
	"strings"
)

// HandleMessage menangani pesan teks dan foto dari user
func (b *BotApp) HandleMessage(update TelegramUpdate) {
	userID := update.Message.From.ID
	chatID := update.Message.Chat.ID
	text := update.Message.Text

	user, err := b.DB.GetOrCreateUser(userID)
	if err != nil {
		return
	}

	// === LOGIKA UPLOAD FOTO ===
	if len(update.Message.Photo) > 0 {
		if user.CurrentState == "uploading_images" {
			b.processPhotoUpload(user, chatID, update)
			return
		} else {
			SendMessage(b.BotToken, chatID, "⚠️ Please click 'Add Image' button first.", nil)
			return
		}
	}

	// === COMMANDS ===
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
			msg := fmt.Sprintf("👤 ID: %d | Credits: %d", user.ID, user.Credits)
			SendMessage(b.BotToken, chatID, msg, nil)
			return
		}
	}

	// === PROMPT TEXT ===
	if user.CurrentState == "waiting_prompt" && user.SelectedModel != "" {
		b.ProcessImageGeneration(user, chatID, text)
		return
	} else if user.CurrentState == "uploading_images" {
		SendMessage(b.BotToken, chatID, "Please click 'Done Uploading' before sending text.", nil)
		return
	}

	SendMessage(b.BotToken, chatID, b.I18n.Get(user.LanguageCode, "use_img_cmd"), nil)
}

// processPhotoUpload menangani logika upload gambar ke Supabase
func (b *BotApp) processPhotoUpload(user *User, chatID int64, update TelegramUpdate) {
	modelConf := b.GetModelByID(user.SelectedModel)

	// Tentukan nama parameter
	paramName := modelConf.ImageParamName
	if paramName == "" {
		if modelConf.AcceptsMultipleImages {
			paramName = "image_input"
		} else {
			paramName = "image"
		}
	}

	// Cek Limit Gambar
	currentCount := 0
	if val, ok := user.DraftConfig[paramName]; ok {
		if list, ok := val.([]interface{}); ok {
			currentCount = len(list)
		} else {
			currentCount = 1
		}
	}

	maxImg := 1
	if modelConf.AcceptsMultipleImages {
		maxImg = 5
	}

	if currentCount >= maxImg {
		SendMessage(b.BotToken, chatID, b.I18n.Get(user.LanguageCode, "upload_limit"), nil)
		return
	}

	// Proses Upload
	SendChatAction(b.BotToken, chatID, "upload_photo")
	photoID := update.Message.Photo[len(update.Message.Photo)-1].FileID

	publicURL, err := b.UploadTelegramToSupabase(photoID, user.ID)
	if err != nil {
		fmt.Printf("[ERROR] Upload failed: %v\n", err)
		SendMessage(b.BotToken, chatID, "❌ Upload failed.", nil)
		return
	}

	// Simpan URL ke Database (Synchronous/Blocking agar data aman)
	if modelConf.AcceptsMultipleImages {
		var currentImages []string
		if val, ok := user.DraftConfig[paramName]; ok {
			if list, ok := val.([]interface{}); ok {
				for _, item := range list {
					if str, ok := item.(string); ok {
						currentImages = append(currentImages, str)
					}
				}
			}
		}
		currentImages = append(currentImages, publicURL)
		// Update DB (Tanpa 'go')
		b.DB.UpdateDraftConfig(user.ID, paramName, currentImages)
	} else {
		// Update DB (Tanpa 'go')
		b.DB.UpdateDraftConfig(user.ID, paramName, publicURL)
	}

	SendMessage(b.BotToken, chatID, b.I18n.Get(user.LanguageCode, "upload_success"), nil)
}

// HandleCallback menangani interaksi tombol
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

	// --- NAVIGATION ---
	if data == "nav_providers" || data == "nav_cancel" {
		go b.DB.ClearState(userID)
		b.ShowProviderList(chatID, user, true, msgID)
		return
	}

	// --- TRIGGER UPLOAD MODE ---
	if data == "trigger_upload" {
		// Update Status Tanpa Reset Config
		go b.DB.UpdateCurrentState(userID, "uploading_images")
		modelConf := b.GetModelByID(user.SelectedModel)
		b.ShowUploadPanel(chatID, msgID, user, modelConf)
		return
	}

	// --- DONE UPLOADING -> BACK TO MAIN ---
	if data == "upload_done" {
		// Update Status Tanpa Reset Config
		go b.DB.UpdateCurrentState(userID, "waiting_prompt")
		
		// Refresh User agar dapat data terbaru
		updatedUser, _ := b.DB.GetOrCreateUser(userID)
		modelConf := b.GetModelByID(updatedUser.SelectedModel)
		b.ShowModelPanel(chatID, msgID, updatedUser, modelConf)
		return
	}

	// --- PROVIDER SELECT ---
	if strings.HasPrefix(data, "prov_") {
		provID := strings.TrimPrefix(data, "prov_")
		var buttons []map[string]string
		for _, m := range b.Models {
			if !m.Enabled {
				continue
			}
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
					"text":          fmt.Sprintf("%s (%d Cr)", m.Name, m.Cost),
					"callback_data": "model_" + m.ID,
				})
			}
		}
		buttons = append(buttons, map[string]string{"text": b.I18n.Get(user.LanguageCode, "back_to_prov"), "callback_data": "nav_providers"})
		msg := b.I18n.Get(user.LanguageCode, "select_model")
		if len(buttons) == 1 {
			msg = b.I18n.Get(user.LanguageCode, "model_unavailable")
		}
		EditMessageText(b.BotToken, chatID, msgID, msg, buttons)
		return
	}

	// --- MODEL SELECT ---
	if strings.HasPrefix(data, "model_") {
		modelID := strings.TrimPrefix(data, "model_")
		modelConf := b.GetModelByID(modelID)
		
		// Gunakan Goroutine untuk update DB (UpdateState mereset draft)
		go func() {
			b.DB.UpdateState(userID, "waiting_prompt", modelID)
			for _, p := range modelConf.Parameters {
				if p.Default != nil {
					b.DB.UpdateDraftConfig(userID, p.Name, p.Default)
				}
			}
		}()
		
		// Simulasi lokal agar UI cepat
		user.DraftConfig = make(map[string]interface{})
		for _, p := range modelConf.Parameters {
			if p.Default != nil {
				user.DraftConfig[p.Name] = p.Default
			}
		}
		b.ShowModelPanel(chatID, msgID, user, modelConf)
		return
	}

	// --- SETTINGS: OPEN SUBMENU ---
	if strings.HasPrefix(data, "set_open|") {
		paramName := strings.TrimPrefix(data, "set_open|")
		modelConf := b.GetModelByID(user.SelectedModel)
		b.ShowSettingOptions(chatID, msgID, user, paramName, modelConf)
		return
	}

	// --- SETTINGS: SET VALUE ---
	if strings.HasPrefix(data, "set_val|") {
		content := strings.TrimPrefix(data, "set_val|")
		parts := strings.SplitN(content, "|", 2)
		if len(parts) == 2 {
			go b.DB.UpdateDraftConfig(userID, parts[0], parts[1])
			user.DraftConfig[parts[0]] = parts[1]
			modelConf := b.GetModelByID(user.SelectedModel)
			b.ShowModelPanel(chatID, msgID, user, modelConf)
		}
		return
	}

	// --- BACK BUTTON ---
	if data == "back_to_panel" {
		modelConf := b.GetModelByID(user.SelectedModel)
		b.ShowModelPanel(chatID, msgID, user, modelConf)
		return
	}
}