package repository

import (
	"context"
	"database/sql"
	"time"

	"vps-store/internal/model"
)

type SettingsRepository struct {
	db *sql.DB
}

func NewSettingsRepository(db *sql.DB) *SettingsRepository {
	return &SettingsRepository{db: db}
}

func (r *SettingsRepository) Get(ctx context.Context) (*model.Settings, error) {
	var s model.Settings
	var networkProvisioned int
	err := r.db.QueryRowContext(ctx,
		`SELECT id, tenancy_ocid, user_ocid, fingerprint, private_key, region,
		        compartment_ocid, vcn_ocid, subnet_ocid, api_base_url, api_token,
		        network_provisioned, created_at, updated_at
		 FROM settings WHERE id = 1`,
	).Scan(
		&s.ID, &s.TenancyOCID, &s.UserOCID, &s.Fingerprint, &s.PrivateKey,
		&s.Region, &s.CompartmentOCID, &s.VCNOCID, &s.SubnetOCID,
		&s.APIBaseURL, &s.APIToken, &networkProvisioned,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	s.NetworkProvisioned = networkProvisioned != 0
	return &s, nil
}

func (r *SettingsRepository) Update(ctx context.Context, s *model.Settings) error {
	networkProvisioned := 0
	if s.NetworkProvisioned {
		networkProvisioned = 1
	}
	s.UpdatedAt = time.Now().UTC()
	_, err := r.db.ExecContext(ctx,
		`UPDATE settings SET
			tenancy_ocid = ?, user_ocid = ?, fingerprint = ?, private_key = ?,
			region = ?, compartment_ocid = ?, vcn_ocid = ?, subnet_ocid = ?,
			api_base_url = ?, api_token = ?, network_provisioned = ?,
			updated_at = ?
		 WHERE id = 1`,
		s.TenancyOCID, s.UserOCID, s.Fingerprint, s.PrivateKey,
		s.Region, s.CompartmentOCID, s.VCNOCID, s.SubnetOCID,
		s.APIBaseURL, s.APIToken, networkProvisioned, s.UpdatedAt,
	)
	return err
}
