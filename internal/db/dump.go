package db

import (
	"database/sql"
	"fmt"
)

func DumpVPS(db *sql.DB) {
	rows, err := db.Query(`SELECT id, display_name, public_ip, ssh_username, ssh_password, initial_credentials, status FROM vps ORDER BY id DESC`)
	if err != nil {
		fmt.Printf("query: %v\n", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		var name, publicIP, sshUser, sshPass, creds, status sql.NullString
		rows.Scan(&id, &name, &publicIP, &sshUser, &sshPass, &creds, &status)
		fmt.Printf("\nVPS %d | %s | %s | %s\n", id, name.String, publicIP.String, status.String)
		if sshUser.Valid { fmt.Printf("  SSH User: %s\n", sshUser.String) }
		if sshPass.Valid { fmt.Printf("  SSH Pass: %s\n", sshPass.String) }
		if creds.Valid   { fmt.Printf("  Creds: %s\n", creds.String) }
	}
}
