CREATE TABLE vps (
    id                    INTEGER PRIMARY KEY AUTOINCREMENT,
    display_name          TEXT NOT NULL,
    template_id           INTEGER REFERENCES templates(id) NOT NULL,
    shape                 TEXT NOT NULL DEFAULT 'VM.Standard.E4.Flex',
    ocpu                  REAL NOT NULL DEFAULT 1.0,
    memory_gb             REAL NOT NULL DEFAULT 8.0,
    boot_volume_size_gb   INTEGER NOT NULL DEFAULT 50,
    oci_instance_id       TEXT,
    public_ip             TEXT,
    private_ip            TEXT,
    status                TEXT NOT NULL DEFAULT 'pending',
    initial_credentials   TEXT,
    created_at            DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at            DATETIME DEFAULT CURRENT_TIMESTAMP
);
