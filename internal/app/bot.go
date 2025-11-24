package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type PhotoSize struct {
	FileID   string `json:"file_id"`
	FileSize int    `json:"file_size"`
}

type TelegramUpdate struct {
	UpdateID int `json:"update_id"`
	Message  struct {
		MessageID int `json:"message_id"`
		From      struct {
			ID       int64  `json:"id"`
			Username string `json:"username"`
		} `json:"from"`
		Text  string      `json:"text"`
		Photo []PhotoSize `json:"photo"`
		Chat  struct {
			ID int64 `json:"id"`
		} `json:"chat"`
	} `json:"message"`
	CallbackQuery struct {
		ID   string `json:"id"`
		From struct {
			ID int64 `json:"id"`
		} `json:"from"`
		Data    string `json:"data"`
		Message struct {
			MessageID int `json:"message_id"`
			Chat      struct {
				ID int64 `json:"id"`
			} `json:"chat"`
		} `json:"message"`
	} `json:"callback_query"`
}

type TelegramResponse struct {
	Ok     bool             `json:"ok"`
	Result []TelegramUpdate `json:"result"`
}

type InputMediaPhoto struct {
	Type      string `json:"type"`
	Media     string `json:"media"`
	Caption   string `json:"caption,omitempty"`
	ParseMode string `json:"parse_mode,omitempty"`
}

// Helper untuk log error dari Telegram API
func checkAPIError(resp *http.Response) error {
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram api error %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

func SendMessage(token string, chatID int64, text string, buttons []map[string]string) error {
	msg := map[string]interface{}{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "HTML", // Ganti ke HTML agar lebih aman dari karakter _
	}
	if len(buttons) > 0 {
		msg["reply_markup"] = buildKeyboard(buttons)
	}
	jsonData, _ := json.Marshal(msg)
	resp, err := http.Post(fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token), "application/json", bytes.NewBuffer(jsonData))
	if err != nil { return err }
	defer resp.Body.Close()
	return checkAPIError(resp)
}

func EditMessageText(token string, chatID int64, messageID int, text string, buttons []map[string]string) error {
	msg := map[string]interface{}{
		"chat_id":    chatID,
		"message_id": messageID,
		"text":       text,
		"parse_mode": "HTML", // Ganti ke HTML
	}
	if len(buttons) > 0 {
		msg["reply_markup"] = buildKeyboard(buttons)
	}
	jsonData, _ := json.Marshal(msg)
	resp, err := http.Post(fmt.Sprintf("https://api.telegram.org/bot%s/editMessageText", token), "application/json", bytes.NewBuffer(jsonData))
	if err != nil { return err }
	defer resp.Body.Close()
	return checkAPIError(resp)
}

func SendPhoto(token string, chatID int64, photoURL string, caption string) error {
	msg := map[string]interface{}{
		"chat_id":    chatID,
		"photo":      photoURL,
		"caption":    caption,
		"parse_mode": "HTML",
	}
	jsonData, _ := json.Marshal(msg)
	resp, err := http.Post(fmt.Sprintf("https://api.telegram.org/bot%s/sendPhoto", token), "application/json", bytes.NewBuffer(jsonData))
	if err != nil { return err }
	defer resp.Body.Close()
	return checkAPIError(resp)
}

func SendMediaGroup(token string, chatID int64, photos []string, caption string) error {
	var mediaGroup []InputMediaPhoto
	for i, url := range photos {
		item := InputMediaPhoto{Type: "photo", Media: url}
		if i == 0 {
			item.Caption = caption
			item.ParseMode = "HTML"
		}
		mediaGroup = append(mediaGroup, item)
	}
	msg := map[string]interface{}{"chat_id": chatID, "media": mediaGroup}
	jsonData, _ := json.Marshal(msg)
	resp, err := http.Post(fmt.Sprintf("https://api.telegram.org/bot%s/sendMediaGroup", token), "application/json", bytes.NewBuffer(jsonData))
	if err != nil { return err }
	defer resp.Body.Close()
	return checkAPIError(resp)
}

func SendChatAction(token string, chatID int64, action string) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendChatAction?chat_id=%d&action=%s", token, chatID, action)
	http.Get(url)
}

func AnswerCallback(token string, callbackID string) {
	http.Get(fmt.Sprintf("https://api.telegram.org/bot%s/answerCallbackQuery?callback_query_id=%s", token, callbackID))
}

func buildKeyboard(buttons []map[string]string) map[string]interface{} {
	var inlineKeyboard [][]interface{}
	row := []interface{}{}
	for i, btn := range buttons {
		row = append(row, btn)
		if (i+1)%2 == 0 || i == len(buttons)-1 {
			inlineKeyboard = append(inlineKeyboard, row)
			row = []interface{}{}
		}
	}
	return map[string]interface{}{"inline_keyboard": inlineKeyboard}
}