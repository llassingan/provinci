package model

import "time"

type Settings struct {
	ID              int64     `json:"id"`
	TenancyOCID     string    `json:"tenancy_ocid"`
	UserOCID        string    `json:"user_ocid"`
	Fingerprint     string    `json:"fingerprint"`
	PrivateKey      string    `json:"-"` // NEVER expose
	Region          string    `json:"region"`
	CompartmentOCID string    `json:"compartment_ocid"`
	APIBaseURL      string    `json:"api_base_url"`
	APIToken        string    `json:"-"` // NEVER expose
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}
