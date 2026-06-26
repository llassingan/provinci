package repository

import (
	"database/sql"
	"os"
	"path/filepath"
	"time"
)

func SeedTemplates(db *sql.DB, projectRoot string) error {
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM templates`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	cloudInitDir := filepath.Join(projectRoot, "ansible", "cloud-init")

	seeds := []struct {
		name        string
		description string
		logoPath    string
		yamlFile    string
		shape       string
		ocpu        float64
		memory      float64
		bootVol     int
	}{
		{"WordPress", "WordPress on Ubuntu with Nginx, PHP-FPM, MySQL", "wordpress", "wordpress.yaml", "VM.Standard.E4.Flex", 1, 8, 50},
		{"Node.js", "Node.js on Ubuntu with Nginx reverse proxy", "nodejs", "nodejs.yaml", "VM.Standard.E4.Flex", 1, 8, 50},
		{"Docker", "Docker on Ubuntu with Docker Compose", "docker", "docker.yaml", "VM.Standard.E4.Flex", 2, 8, 50},
		{"Ubuntu", "Vanilla Ubuntu with essential tools", "ubuntu", "ubuntu.yaml", "VM.Standard.E4.Flex", 1, 4, 30},
	}

	for _, s := range seeds {
		yamlPath := filepath.Join(cloudInitDir, s.yamlFile)
		yamlContent, err := os.ReadFile(yamlPath)
		if err != nil {
			return err
		}

		now := time.Now().UTC()
		_, err = db.Exec(
			`INSERT INTO templates (name, description, type, logo_url, cloud_init_yaml,
			 shape, default_ocpu, default_memory, boot_volume_size_gb, created_at)
			 VALUES (?, ?, 'predefined', ?, ?, ?, ?, ?, ?, ?)`,
			s.name, s.description, s.logoPath, string(yamlContent),
			s.shape, s.ocpu, s.memory, s.bootVol, now,
		)
		if err != nil {
			return err
		}
	}

	return nil
}
