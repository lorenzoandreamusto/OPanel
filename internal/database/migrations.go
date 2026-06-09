package database

import (
	"fmt"
)

var migrations = []string{
	`CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		email TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		role TEXT NOT NULL DEFAULT 'user' CHECK(role IN ('admin', 'user')),
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`,
	`CREATE TABLE IF NOT EXISTS refresh_tokens (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		token_hash TEXT NOT NULL,
		expires_at DATETIME NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	)`,
	`CREATE TABLE IF NOT EXISTS domains (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE NOT NULL,
		ip_address TEXT NOT NULL DEFAULT '0.0.0.0',
		status TEXT NOT NULL DEFAULT 'pending' CHECK(status IN ('active', 'suspended', 'pending')),
		owner_id INTEGER NOT NULL,
		document_root TEXT NOT NULL DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE CASCADE
	)`,
	`CREATE TABLE IF NOT EXISTS databases (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE NOT NULL,
		owner_id INTEGER NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE CASCADE
	)`,
	`CREATE TABLE IF NOT EXISTS database_users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT NOT NULL,
		host TEXT NOT NULL DEFAULT '%',
		database_id INTEGER NOT NULL,
		privileges TEXT NOT NULL DEFAULT 'ALL PRIVILEGES',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (database_id) REFERENCES databases(id) ON DELETE CASCADE,
		UNIQUE(username, host, database_id)
	)`,
	`ALTER TABLE domains ADD COLUMN php_version TEXT NOT NULL DEFAULT '8.4'`,
	`ALTER TABLE domains ADD COLUMN hosting_type TEXT NOT NULL DEFAULT 'php' CHECK(hosting_type IN ('static', 'php'))`,
	`ALTER TABLE domains ADD COLUMN ssl_enabled INTEGER NOT NULL DEFAULT 0`,
	`ALTER TABLE domains ADD COLUMN auto_db INTEGER NOT NULL DEFAULT 0`,
}

func (d *DB) RunMigrations() error {
	_, err := d.Exec(`CREATE TABLE IF NOT EXISTS migrations (
		id INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	for i, m := range migrations {
		var count int
		err := d.QueryRow("SELECT COUNT(*) FROM migrations WHERE id = ?", i+1).Scan(&count)
		if err != nil {
			return fmt.Errorf("failed to check migration %d: %w", i+1, err)
		}
		if count > 0 {
			continue
		}

		tx, err := d.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin migration transaction: %w", err)
		}

		if _, err := tx.Exec(m); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to apply migration %d: %w", i+1, err)
		}

		if _, err := tx.Exec("INSERT INTO migrations (id, name) VALUES (?, ?)", i+1, fmt.Sprintf("migration_%d", i+1)); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to record migration %d: %w", i+1, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration %d: %w", i+1, err)
		}

		fmt.Printf("Applied migration %d\n", i+1)
	}

	return nil
}
