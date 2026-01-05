package models

import "time"

// EmailTemplate represents an email template
type EmailTemplate struct {
	ID              int       `json:"id"`
	Slug            string    `json:"slug"`
	SubjectTemplate string    `json:"subject_template"`
	BodyTemplate    string    `json:"body_template"`
	CreatedAt       time.Time `json:"created_at,omitempty"`
	UpdatedAt       time.Time `json:"updated_at,omitempty"`
}

// CreateUpdateEmailTemplateRequest represents a request to create or update an email template
type CreateUpdateEmailTemplateRequest struct {
	Slug            string `json:"slug,omitempty"`
	SubjectTemplate string `json:"subject_template,omitempty"`
	BodyTemplate    string `json:"body_template,omitempty"`
}

// EmailTemplateListItem represents an email template in a list response
type EmailTemplateListItem struct {
	ID   int    `json:"id"`
	Slug string `json:"slug"`
}

// EmailTemplateTemplate represents an email template template
type EmailTemplateParts struct {
	SubjectTemplate string `json:"subject_template"`
	BodyTemplate    string `json:"body_template"`
}
