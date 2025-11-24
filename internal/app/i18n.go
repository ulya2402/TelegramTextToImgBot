package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type I18nManager struct {
	translations map[string]map[string]string
	mu           sync.RWMutex
}

func NewI18nManager() *I18nManager {
	return &I18nManager{
		translations: make(map[string]map[string]string),
	}
}

func (m *I18nManager) LoadTranslations(dir string) error {
	files, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, f := range files {
		if filepath.Ext(f.Name()) == ".json" {
			langCode := f.Name()[0 : len(f.Name())-5]
			content, err := os.ReadFile(filepath.Join(dir, f.Name()))
			if err != nil {
				fmt.Printf("[ERROR] Failed to read locale file %s: %v\n", f.Name(), err)
				continue
			}

			var data map[string]string
			if err := json.Unmarshal(content, &data); err != nil {
				fmt.Printf("[ERROR] Failed to parse locale file %s: %v\n", f.Name(), err)
				continue
			}
			m.translations[langCode] = data
			fmt.Printf("[INFO] Loaded locale: %s\n", langCode)
		}
	}
	return nil
}

func (m *I18nManager) Get(lang, key string, args ...interface{}) string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if data, ok := m.translations[lang]; ok {
		if val, ok := data[key]; ok {
			if len(args) > 0 {
				return fmt.Sprintf(val, args...)
			}
			return val
		}
	}
	
	if data, ok := m.translations["en"]; ok {
		if val, ok := data[key]; ok {
			if len(args) > 0 {
				return fmt.Sprintf(val, args...)
			}
			return val
		}
	}
	
	return key
}