package model

import "time"

type Domain struct {
	ID           int       `json:"id"`
	Name         string    `json:"name"`
	IPAddress    string    `json:"ip_address"`
	Status       string    `json:"status"`
	OwnerID      int       `json:"owner_id"`
	DocumentRoot string    `json:"document_root"`
	PHPVersion   string    `json:"php_version"`
	HostingType  string    `json:"hosting_type"`
	SSLEnabled   bool      `json:"ssl_enabled"`
	AutoDB       bool      `json:"auto_db"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type CreateDomainRequest struct {
	Name        string `json:"name"`
	PHPVersion  string `json:"php_version,omitempty"`
	HostingType string `json:"hosting_type,omitempty"`
	SSLEnabled  bool   `json:"ssl_enabled,omitempty"`
	AutoDB      bool   `json:"auto_db,omitempty"`
}

type UpdateDomainRequest struct {
	Status      string `json:"status,omitempty"`
	PHPVersion  string `json:"php_version,omitempty"`
	HostingType string `json:"hosting_type,omitempty"`
	SSLEnabled  *bool  `json:"ssl_enabled,omitempty"`
	AutoDB      *bool  `json:"auto_db,omitempty"`
}
