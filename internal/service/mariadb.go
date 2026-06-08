package service

import (
	"database/sql"
	"fmt"
	"log/slog"

	"opanel/internal/database"

	_ "github.com/go-sql-driver/mysql"
)

type MariaDBService struct {
	db *database.DB
}

func NewMariaDBService(db *database.DB) *MariaDBService {
	return &MariaDBService{db: db}
}

// ConnectMariaDB opens a connection to the local MariaDB instance.
// Uses root with no password by default (Plesk-style setup).
func (s *MariaDBService) connectMariaDB() (*sql.DB, error) {
	dsn := "root@unix(/var/run/mysqld/mysqld.sock)/"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MariaDB: %w", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping MariaDB: %w", err)
	}
	return db, nil
}

// CreateDatabase creates a database in MariaDB.
func (s *MariaDBService) CreateDatabase(name string) error {
	mdb, err := s.connectMariaDB()
	if err != nil {
		return err
	}
	defer mdb.Close()

	_, err = mdb.Exec("CREATE DATABASE `" + name + "`")
	if err != nil {
		return fmt.Errorf("failed to create MariaDB database: %w", err)
	}

	slog.Info("MariaDB database created", "name", name)
	return nil
}

// DropDatabase drops a database from MariaDB.
func (s *MariaDBService) DropDatabase(name string) error {
	mdb, err := s.connectMariaDB()
	if err != nil {
		return err
	}
	defer mdb.Close()

	_, err = mdb.Exec("DROP DATABASE `" + name + "`")
	if err != nil {
		return fmt.Errorf("failed to drop MariaDB database: %w", err)
	}

	slog.Info("MariaDB database dropped", "name", name)
	return nil
}

// CreateUser creates a user in MariaDB with the given host.
func (s *MariaDBService) CreateUser(username, host, password string) error {
	mdb, err := s.connectMariaDB()
	if err != nil {
		return err
	}
	defer mdb.Close()

	query := fmt.Sprintf("CREATE USER `%s`@`%s` IDENTIFIED BY '%s'", username, host, escapeString(password))
	_, err = mdb.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to create MariaDB user: %w", err)
	}

	slog.Info("MariaDB user created", "username", username, "host", host)
	return nil
}

// DropUser removes a user from MariaDB.
func (s *MariaDBService) DropUser(username, host string) error {
	mdb, err := s.connectMariaDB()
	if err != nil {
		return err
	}
	defer mdb.Close()

	_, err = mdb.Exec("DROP USER `" + username + "`@`" + host + "`")
	if err != nil {
		return fmt.Errorf("failed to drop MariaDB user: %w", err)
	}

	slog.Info("MariaDB user dropped", "username", username, "host", host)
	return nil
}

// ChangePassword changes a user's password in MariaDB.
func (s *MariaDBService) ChangePassword(username, host, password string) error {
	mdb, err := s.connectMariaDB()
	if err != nil {
		return err
	}
	defer mdb.Close()

	query := fmt.Sprintf("ALTER USER `%s`@`%s` IDENTIFIED BY '%s'", username, host, escapeString(password))
	_, err = mdb.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to change MariaDB user password: %w", err)
	}

	slog.Info("MariaDB user password changed", "username", username, "host", host)
	return nil
}

// GrantPrivileges grants privileges on a database to a user in MariaDB.
// First revokes all existing privileges on the database, then grants the new ones.
func (s *MariaDBService) GrantPrivileges(username, host, database, privileges string) error {
	mdb, err := s.connectMariaDB()
	if err != nil {
		return err
	}
	defer mdb.Close()

	// First revoke all privileges on this database
	revokeQuery := fmt.Sprintf("REVOKE ALL PRIVILEGES ON `%s`.* FROM `%s`@`%s`", database, username, host)
	_, err = mdb.Exec(revokeQuery)
	if err != nil {
		// Ignore error - user may not have any privileges yet
		slog.Debug("REVOKE before GRANT (non-fatal)", "error", err)
	}

	if privileges == "" {
		privileges = "ALL PRIVILEGES"
	}

	query := fmt.Sprintf("GRANT %s ON `%s`.* TO `%s`@`%s`", privileges, database, username, host)
	_, err = mdb.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to grant MariaDB privileges: %w", err)
	}

	_, err = mdb.Exec("FLUSH PRIVILEGES")
	if err != nil {
		slog.Warn("Failed to flush privileges (non-fatal)", "error", err)
	}

	slog.Info("MariaDB privileges granted", "username", username, "host", host, "database", database, "privileges", privileges)
	return nil
}

// escapeString escapes single quotes for SQL string literals
func escapeString(s string) string {
	s = replaceAll(s, "'", "''")
	s = replaceAll(s, "\\", "\\\\")
	return s
}

func replaceAll(s, old, new string) string {
	result := ""
	for i := 0; i < len(s); {
		if i+len(old) <= len(s) && s[i:i+len(old)] == old {
			result += new
			i += len(old)
		} else {
			result += string(s[i])
			i++
		}
	}
	return result
}
