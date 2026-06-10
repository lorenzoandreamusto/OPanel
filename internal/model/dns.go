package model

import "time"

// DNSZone represents a DNS zone for a domain
type DNSZone struct {
	ID        int       `json:"id"`
	DomainID  int       `json:"domain_id"`
	Name      string    `json:"name"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// DNSRecord represents a DNS record within a zone
type DNSRecord struct {
	ID       int    `json:"id"`
	ZoneID   int    `json:"zone_id"`
	Type     string `json:"type"`
	Name     string `json:"name"`
	Value    string `json:"value"`
	TTL      int    `json:"ttl"`
	Priority int    `json:"priority,omitempty"`
	Enabled  bool   `json:"enabled"`
}

type CreateDNSZoneRequest struct {
	DomainID int `json:"domain_id"`
}

type CreateDNSRecordRequest struct {
	Type     string `json:"type"`
	Name     string `json:"name"`
	Value    string `json:"value"`
	TTL      int    `json:"ttl,omitempty"`
	Priority int    `json:"priority,omitempty"`
}

type UpdateDNSRecordRequest struct {
	Type     string `json:"type,omitempty"`
	Name     string `json:"name,omitempty"`
	Value    string `json:"value,omitempty"`
	TTL      int    `json:"ttl,omitempty"`
	Priority *int   `json:"priority,omitempty"`
	Enabled  *bool  `json:"enabled,omitempty"`
}
