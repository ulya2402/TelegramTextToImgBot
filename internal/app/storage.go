package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"time"
)

// Nama Bucket didefinisikan sebagai konstanta agar konsisten
const BucketName = "bot-uploads"

type TelegramFileResponse struct {
	Ok     bool `json:"ok"`
	Result struct {
		FileID   string `json:"file_id"`
		FilePath string `json:"file_path"`
	} `json:"result"`
}

// FUNGSI BARU: Cek & Buat Bucket Otomatis
func (b *BotApp) EnsureBucketExists() error {
	client := &http.Client{Timeout: 10 * time.Second}
	
	// 1. Cek apakah bucket sudah ada (GET /storage/v1/bucket/{id})
	checkURL := fmt.Sprintf("%s/storage/v1/bucket/%s", b.SupabaseURL, BucketName)
	reqCheck, _ := http.NewRequest("GET", checkURL, nil)
	reqCheck.Header.Set("Authorization", "Bearer "+b.SupabaseKey)
	
	respCheck, err := client.Do(reqCheck)
	if err != nil {
		return fmt.Errorf("failed to check bucket: %v", err)
	}
	defer respCheck.Body.Close()

	if respCheck.StatusCode == 200 {
		fmt.Println("[INFO] Storage bucket 'bot-uploads' exists.")
		return nil
	}

	// 2. Jika belum ada (404), Buat Bucket Baru (POST /storage/v1/bucket)
	fmt.Println("[INFO] Creating storage bucket 'bot-uploads'...")
	
	createURL := fmt.Sprintf("%s/storage/v1/bucket", b.SupabaseURL)
	payload := map[string]interface{}{
		"id":     BucketName,
		"name":   BucketName,
		"public": true, // PENTING: Harus Public agar Replicate bisa baca
	}
	jsonData, _ := json.Marshal(payload)

	reqCreate, _ := http.NewRequest("POST", createURL, bytes.NewBuffer(jsonData))
	reqCreate.Header.Set("Authorization", "Bearer "+b.SupabaseKey)
	reqCreate.Header.Set("Content-Type", "application/json")

	respCreate, err := client.Do(reqCreate)
	if err != nil {
		return fmt.Errorf("failed to create bucket: %v", err)
	}
	defer respCreate.Body.Close()

	if respCreate.StatusCode != 200 {
		body, _ := io.ReadAll(respCreate.Body)
		return fmt.Errorf("create bucket error %d: %s", respCreate.StatusCode, string(body))
	}

	fmt.Println("[INFO] Storage bucket created successfully.")
	return nil
}

func (b *BotApp) UploadTelegramToSupabase(fileID string, userID int64) (string, error) {
	// 1. Get File Path
	client := &http.Client{Timeout: 30 * time.Second}
	urlInfo := fmt.Sprintf("https://api.telegram.org/bot%s/getFile?file_id=%s", b.BotToken, fileID)
	
	respInfo, err := client.Get(urlInfo)
	if err != nil { return "", fmt.Errorf("get file info failed: %v", err) }
	defer respInfo.Body.Close()

	var fileData TelegramFileResponse
	json.NewDecoder(respInfo.Body).Decode(&fileData)
	if !fileData.Ok { return "", fmt.Errorf("telegram api error") }

	// 2. Download Content
	urlContent := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", b.BotToken, fileData.Result.FilePath)
	respContent, err := client.Get(urlContent)
	if err != nil { return "", fmt.Errorf("download content failed: %v", err) }
	defer respContent.Body.Close()

	fileBytes, err := io.ReadAll(respContent.Body)
	if err != nil { return "", err }

	// 3. Filename
	ext := filepath.Ext(fileData.Result.FilePath)
	if ext == "" { ext = ".jpg" }
	filename := fmt.Sprintf("%d_%d%s", userID, time.Now().UnixNano(), ext)

	// 4. Upload ke Supabase (Menggunakan Constant BucketName)
	supabaseStorageURL := fmt.Sprintf("%s/storage/v1/object/%s/%s", b.SupabaseURL, BucketName, filename)
	
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", filename)
	part.Write(fileBytes)
	writer.Close()

	req, _ := http.NewRequest("POST", supabaseStorageURL, body)
	req.Header.Set("Authorization", "Bearer "+b.SupabaseKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	respUpload, err := client.Do(req)
	if err != nil { return "", fmt.Errorf("upload failed: %v", err) }
	defer respUpload.Body.Close()

	if respUpload.StatusCode != 200 {
		body, _ := io.ReadAll(respUpload.Body)
		return "", fmt.Errorf("supabase upload error %d: %s", respUpload.StatusCode, string(body))
	}

	// 5. Public URL
	publicURL := fmt.Sprintf("%s/storage/v1/object/public/%s/%s", b.SupabaseURL, BucketName, filename)
	return publicURL, nil
}