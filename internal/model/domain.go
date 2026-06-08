package model

import "time"

type Domain struct {
	ID           int       `json:"id"`
	Name         string    `json:"name"`
	IPAddress    string    `json:"ip_address"`
	Status       string    `json:"status"`
	OwnerID      int       `json:"owner_id"`
	DocumentRoot string    `json:"document_root"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type CreateDomainRequest struct {
	Name string `json:"name"`
}

type UpdateDomainRequest struct {
	Status string `json:"status,omitempty"`
}
