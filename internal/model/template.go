package model

import "time"

type Template struct {
	ID               int64     `json:"id"`
	Name             string    `json:"name"`
	Description      string    `json:"description"`
	Type             string    `json:"type"` // "predefined" | "custom"
	LogoURL          string    `json:"logo_url,omitempty"`
	CloudInitYAML    string    `json:"-"` // never sent to frontend
	Shape            string    `json:"shape"`
	DefaultOCPU      float64   `json:"default_ocpu"`
	DefaultMemory    float64   `json:"default_memory"`
	BootVolumeSizeGB int       `json:"boot_volume_size_gb"`
	CreatedAt        time.Time `json:"created_at"`
}
