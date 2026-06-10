package model

import "time"

// MailAccount represents an email account
type MailAccount struct {
	ID        int       `json:"id"`
	DomainID  int       `json:"domain_id"`
	Username  string    `json:"username"`
	Password  string    `json:"password,omitempty"`
	Quota     int64     `json:"quota"`
	Used      int64     `json:"used"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// MailDomain represents a mail-enabled domain
type MailDomain struct {
	ID           int       `json:"id"`
	DomainID     int       `json:"domain_id"`
	Name         string    `json:"name"`
	Enabled      bool      `json:"enabled"`
	DKIMEnabled  bool      `json:"dkim_enabled"`
	DKIMSelector string    `json:"dkim_selector,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type CreateMailDomainRequest struct {
	DomainID int `json:"domain_id"`
}

type CreateMailAccountRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Quota    int64  `json:"quota,omitempty"`
}

type UpdateMailAccountRequest struct {
	Password string `json:"password,omitempty"`
	Quota    *int64 `json:"quota,omitempty"`
	Enabled  *bool  `json:"enabled,omitempty"`
}

type MailAutoconfigResponse struct {
	Domain    string `json:"domain"`
	IMAPHost  string `json:"imap_host"`
	IMAPPort  int    `json:"imap_port"`
	IMAPSSL   int    `json:"imap_ssl"`
	IMAPTLS   int    `json:"imap_starttls"`
	SMTPHost  string `json:"smtp_host"`
	SMTPPort  int    `json:"smtp_port"`
	SMTPSSL   int    `json:"smtp_ssl"`
	SMTPTLS   int    `json:"smtp_starttls"`
	SMTPAuth  bool   `json:"smtp_auth"`
	Username  string `json:"username"`
}
