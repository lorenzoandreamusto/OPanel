package model

import "time"

type Database struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	OwnerID   int       `json:"owner_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type DatabaseUser struct {
	ID         int       `json:"id"`
	Username   string    `json:"username"`
	Host       string    `json:"host"`
	DatabaseID int       `json:"database_id"`
	Privileges string    `json:"privileges"`
	CreatedAt  time.Time `json:"created_at"`
}

type CreateDatabaseRequest struct {
	Name string `json:"name"`
}

type CreateDatabaseUserRequest struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	DatabaseID int    `json:"database_id"`
	Privileges string `json:"privileges,omitempty"`
}

type UpdateDatabaseUserRequest struct {
	Password   string `json:"password,omitempty"`
	Privileges string `json:"privileges,omitempty"`
}
