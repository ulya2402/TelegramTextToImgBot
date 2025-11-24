package app

import (
	"fmt"
	"strings"
)

func (b *BotApp) ShowProviderList(chatID int64, user *User, isEdit bool, msgID int) {
	var buttons []map[string]string
	for _, p := range b.Providers {
		buttons = append(buttons, map[string]string{"text": p.Name, "callback_data": "prov_" + p.ID})
	}
	text := b.I18n.Get(user.LanguageCode, "select_provider")
	if isEdit {
		EditMessageText(b.BotToken, chatID, msgID, text, buttons)
	} else {
		SendMessage(b.BotToken, chatID, text, buttons)
	}
}

func (b *BotApp) ShowModelPanel(chatID int64, msgID int, user *User, modelConf ModelConfig) error {
	settingText := ""
	imgCount := 0

	// Cek jumlah gambar
	paramName := modelConf.ImageParamName
	if paramName == "" {
		if modelConf.AcceptsMultipleImages { paramName = "image_input" } else { paramName = "image" }
	}

	if val, ok := user.DraftConfig[paramName]; ok {
		if list, ok := val.([]interface{}); ok {
			imgCount = len(list)
		} else if _, ok := val.(string); ok {
			imgCount = 1
		}
	}

	for k, v := range user.DraftConfig {
		if k == paramName { continue } 
		cleanKey := strings.ReplaceAll(k, "_", " ")
		settingText += fmt.Sprintf("\n• <b>%s:</b> %v", cleanKey, v)
	}
	
	totalCost := b.CalculateTotalCost(modelConf.Cost, user.DraftConfig)
	settingText += fmt.Sprintf("\n\n💰 <b>Cost:</b> %d Credits", totalCost)

	panelText := fmt.Sprintf("🤖 <b>%s</b>\n\nCurrent Settings:%s\n\n👇 <i>Tap buttons to configure, OR type prompt to start:</i>", modelConf.Name, settingText)

	var buttons []map[string]string
	
	// BUTTON ADD IMAGE (Sekarang sudah dikenali karena types.go sudah diupdate)
	if modelConf.AcceptsImageInput {
		maxImg := 1
		if modelConf.AcceptsMultipleImages { maxImg = 5 }
		
		btnText := fmt.Sprintf(b.I18n.Get(user.LanguageCode, "btn_add_image"), imgCount, maxImg)
		buttons = append(buttons, map[string]string{
			"text": btnText,
			"callback_data": "trigger_upload",
		})
	}

	for _, p := range modelConf.Parameters {
		if len(p.Options) > 0 {
			label := p.Label
			if label == "" { label = p.Name }
			buttons = append(buttons, map[string]string{
				"text": b.I18n.Get(user.LanguageCode, "change_btn", label),
				"callback_data": fmt.Sprintf("set_open|%s", p.Name),
			})
		}
	}

	buttons = append(buttons, map[string]string{
		"text": b.I18n.Get(user.LanguageCode, "cancel_btn"),
		"callback_data": "nav_cancel",
	})

	return EditMessageText(b.BotToken, chatID, msgID, panelText, buttons)
}

func (b *BotApp) ShowUploadPanel(chatID int64, msgID int, user *User, modelConf ModelConfig) {
	maxImg := 1
	if modelConf.AcceptsMultipleImages { maxImg = 5 }

	currentCount := 0
	paramName := modelConf.ImageParamName
	if paramName == "" {
		if modelConf.AcceptsMultipleImages { paramName = "image_input" } else { paramName = "image" }
	}
	if val, ok := user.DraftConfig[paramName]; ok {
		if list, ok := val.([]interface{}); ok {
			currentCount = len(list)
		} else if _, ok := val.(string); ok {
			currentCount = 1
		}
	}

	text := fmt.Sprintf(b.I18n.Get(user.LanguageCode, "upload_mode_msg"), maxImg, currentCount)
	
	var buttons []map[string]string
	buttons = append(buttons, map[string]string{
		"text": b.I18n.Get(user.LanguageCode, "btn_done_img"),
		"callback_data": "upload_done",
	})

	EditMessageText(b.BotToken, chatID, msgID, text, buttons)
}

func (b *BotApp) ShowSettingOptions(chatID int64, msgID int, user *User, paramName string, modelConf ModelConfig) {
	var targetParam ModelParameter
	for _, p := range modelConf.Parameters {
		if p.Name == paramName {
			targetParam = p
			break
		}
	}
	text := "Select value for <b>" + targetParam.Label + "</b>:"
	var buttons []map[string]string
	for _, opt := range targetParam.Options {
		valStr := fmt.Sprintf("%v", opt)
		buttons = append(buttons, map[string]string{
			"text": valStr,
			"callback_data": fmt.Sprintf("set_val|%s|%s", paramName, valStr),
		})
	}
	buttons = append(buttons, map[string]string{
		"text": b.I18n.Get(user.LanguageCode, "back_btn"),
		"callback_data": "back_to_panel",
	})
	EditMessageText(b.BotToken, chatID, msgID, text, buttons)
}