CREATE TABLE networks (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT NOT NULL,
    cidr_vcn    TEXT NOT NULL,
    cidr_subnet TEXT NOT NULL,
    vcn_ocid    TEXT DEFAULT '',
    subnet_ocid TEXT DEFAULT '',
    status      TEXT NOT NULL DEFAULT 'pending',
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE vps ADD COLUMN network_id INTEGER REFERENCES networks(id);

ALTER TABLE settings RENAME COLUMN vcn_ocid TO _vcn_ocid_deprecated;
ALTER TABLE settings RENAME COLUMN subnet_ocid TO _subnet_ocid_deprecated;
ALTER TABLE settings RENAME COLUMN network_provisioned TO _network_provisioned_deprecated;
