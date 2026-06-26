CREATE TABLE templates (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    name            TEXT UNIQUE NOT NULL,
    description     TEXT NOT NULL,
    type            TEXT NOT NULL DEFAULT 'predefined',
    logo_url        TEXT,
    cloud_init_yaml TEXT NOT NULL,
    shape           TEXT NOT NULL DEFAULT 'VM.Standard.E4.Flex',
    default_ocpu    REAL NOT NULL DEFAULT 1.0,
    default_memory  REAL NOT NULL DEFAULT 8.0,
    boot_volume_size_gb INTEGER NOT NULL DEFAULT 50,
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP
);
