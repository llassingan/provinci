package model

import (
	"database/sql"
	"time"
)

type VPS struct {
	ID                 int64          `json:"id"`
	DisplayName        string         `json:"display_name"`
	TemplateID         int64          `json:"template_id"`
	NetworkID          sql.NullInt64  `json:"network_id"`
	Shape              string         `json:"shape"`
	OCPU               float64        `json:"ocpu"`
	MemoryGB           float64        `json:"memory_gb"`
	BootVolumeSizeGB   int            `json:"boot_volume_size_gb"`
	OCIInstanceID      sql.NullString `json:"oci_instance_id"`
	PublicIP           sql.NullString `json:"public_ip"`
	PrivateIP          sql.NullString `json:"private_ip"`
	Status             string         `json:"status"`
	InitialCredentials sql.NullString `json:"initial_credentials"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
}
