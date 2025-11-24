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
	for k, v := range user.DraftConfig {
		cleanKey := strings.ReplaceAll(k, "_", " ")
		if k == "image_input" {
			settingText += fmt.Sprintf("\n• <b>Image:</b> Attached ✅")
		} else {
			settingText += fmt.Sprintf("\n• <b>%s:</b> %v", cleanKey, v)
		}
	}
	
	totalCost := b.CalculateTotalCost(modelConf.Cost, user.DraftConfig)
	settingText += fmt.Sprintf("\n\n💰 <b>Cost:</b> %d Credits", totalCost)

	panelText := fmt.Sprintf("🤖 <b>%s</b>\n\nCurrent Settings:%s\n\n👇 <i>Tap buttons to change settings, OR type prompt to start:</i>", modelConf.Name, settingText)

	var buttons []map[string]string
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