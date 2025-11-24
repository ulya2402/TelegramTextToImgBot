package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type ReplicateConfig struct {
	Token string
}

type ReplicateRequest struct {
	Input map[string]interface{} `json:"input"`
}

type ReplicateResponse struct {
	ID     string      `json:"id"`
	Status string      `json:"status"`
	Output interface{} `json:"output"`
	Error  interface{} `json:"error"`
	URLs   struct {
		Get string `json:"get"`
	} `json:"urls"`
}

func NewReplicate(token string) *ReplicateConfig {
	return &ReplicateConfig{Token: token}
}

func (r *ReplicateConfig) Generate(modelConf ModelConfig, userInput string, extraInputs map[string]interface{}) ([]string, error) {
	client := &http.Client{Timeout: 120 * time.Second}
	payloadData := make(map[string]interface{})

	// 1. Masukkan Default Parameters
	for _, param := range modelConf.Parameters {
		if param.Default != nil {
			payloadData[param.Name] = param.Default
		}
	}

	// 2. Masukkan Input User
	for k, v := range extraInputs {
		// KASUS A: Input adalah Array (Multiple Images)
		if list, ok := v.([]interface{}); ok {
			var rawURLs []string
			for _, item := range list {
				if str, ok := item.(string); ok {
					// Validasi sederhana: harus ada http
					if strings.HasPrefix(str, "http") {
						rawURLs = append(rawURLs, str)
					}
				}
			}
			// LANGSUNG masukkan sebagai Array of Strings.
			// Tidak perlu dibungkus object {"value":...} lagi.
			payloadData[k] = rawURLs
			continue
		}

		// KASUS B: Input adalah String Tunggal
		if strVal, isString := v.(string); isString {
			// Validasi URL untuk gambar
			isImageParam := (k == "image_input" || k == "input_images" || k == "reference_images" || k == "image")
			if isImageParam && !strings.HasPrefix(strVal, "http") {
				return nil, fmt.Errorf("invalid image URL: %s", strVal)
			}

			// JIKA parameter ini biasanya butuh Array (seperti image_input), kita bungkus string tunggal jadi Array.
			if k == "image_input" || k == "input_images" || k == "reference_images" {
				payloadData[k] = []string{strVal}
				continue
			}

			// Type Casting untuk Angka
			var expectedType string
			for _, p := range modelConf.Parameters {
				if p.Name == k {
					expectedType = p.Type
					break
				}
			}

			if expectedType == "integer" {
				if intVal, err := strconv.Atoi(strVal); err == nil {
					payloadData[k] = intVal
				} else {
					payloadData[k] = v
				}
			} else if expectedType == "number" || expectedType == "float" {
				if floatVal, err := strconv.ParseFloat(strVal, 64); err == nil {
					payloadData[k] = floatVal
				} else {
					payloadData[k] = v
				}
			} else {
				// String biasa (misal aspect_ratio)
				payloadData[k] = v
			}
		} else {
			// Boolean atau tipe lain
			payloadData[k] = v
		}
	}

	// 3. Set Prompt
	payloadData["prompt"] = userInput

	// --- DEBUGGING LOG ---
	debugJSON, _ := json.MarshalIndent(payloadData, "", "  ")
	fmt.Printf("[DEBUG] Payload to Replicate:\n%s\n", string(debugJSON))
	// ---------------------

	reqBody := ReplicateRequest{Input: payloadData}
	
	// Parsing ID
	var apiURL string
	if strings.HasPrefix(modelConf.ReplicateID, "google/") {
		parts := strings.Split(modelConf.ReplicateID, "/")
		apiURL = fmt.Sprintf("https://api.replicate.com/v1/models/%s/%s/predictions", parts[0], parts[1])
	} else {
		parts := strings.Split(modelConf.ReplicateID, "/")
		if len(parts) == 2 {
			apiURL = fmt.Sprintf("https://api.replicate.com/v1/models/%s/%s/predictions", parts[0], parts[1])
		} else {
			return nil, fmt.Errorf("invalid replicate_id format")
		}
	}

	jsonData, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", "Bearer "+r.Token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Prefer", "wait=55") 

	resp, err := client.Do(req)
	if err != nil { return nil, err }
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("api status %d: %s", resp.StatusCode, string(body))
	}

	bodyBytes, _ := io.ReadAll(resp.Body)
	var result ReplicateResponse
	json.Unmarshal(bodyBytes, &result)

	if result.Error != nil { return nil, fmt.Errorf("%v", result.Error) }

	if result.Status == "succeeded" { return parseOutput(result.Output), nil }

	return r.pollResult(result.URLs.Get)
}

func (r *ReplicateConfig) pollResult(url string) ([]string, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	for {
		time.Sleep(2 * time.Second)
		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set("Authorization", "Bearer "+r.Token)
		resp, err := client.Do(req)
		if err != nil { return nil, err }
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		var result ReplicateResponse
		json.Unmarshal(bodyBytes, &result)
		
		fmt.Printf("[INFO] Polling: %s\n", result.Status)
		if result.Status == "succeeded" { return parseOutput(result.Output), nil }
		if result.Status == "failed" || result.Status == "canceled" { return nil, fmt.Errorf("failed") }
	}
}

func parseOutput(output interface{}) []string {
	var urls []string
	switch v := output.(type) {
	case string:
		urls = append(urls, v)
	case []interface{}:
		for _, item := range v {
			if str, ok := item.(string); ok {
				urls = append(urls, str)
			}
		}
	}
	return urls
}