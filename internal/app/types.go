package app

type Provider struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type ModelParameter struct {
	Name    string        `json:"name"`
	Label   string        `json:"label"`
	Default interface{}   `json:"default"`
	Type    string        `json:"type"`
	Options []interface{} `json:"options"`
}

type ModelConfig struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Type        string           `json:"type"`
	ReplicateID string           `json:"replicate_id"`
	Cost        int              `json:"cost"`
	Enabled     bool             `json:"enabled"`
	Parameters  []ModelParameter `json:"parameters"`
	Description string           `json:"description"`

	// --- FIELD BARU (PENTING) ---
	// Field ini wajib ada agar ui.go dan handlers.go tidak error
	AcceptsImageInput     bool   `json:"accepts_image_input"`
	AcceptsMultipleImages bool   `json:"accepts_multiple_images"`
	ImageParamName        string `json:"image_parameter_name"` 
}