package kyc

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Moderation states for a KYC submission. A submission is created pending and an
// admin approves or rejects it.
const (
	StatusPending  = "pending"
	StatusApproved = "approved"
	StatusRejected = "rejected"
)

// Derived profile states surfaced to the customer. A user with no submission is
// "unverified"; an approved submission reads as "verified".
const (
	ProfileUnverified = "unverified"
	ProfilePending    = "pending"
	ProfileVerified   = "verified"
	ProfileRejected   = "rejected"
)

// Accepted identity-document types.
const (
	DocPassport = "passport"
	DocIDCard   = "id_card"
	DocLicense  = "license"
)

// Submission is a customer's KYC record (one per user). It carries personal
// background info only — no document images (per product scope).
type Submission struct {
	ID               bson.ObjectID `bson:"_id,omitempty"             json:"id"`
	UserID           bson.ObjectID `bson:"userId"                    json:"userId"`
	FullName         string        `bson:"fullName"                  json:"fullName"`
	DateOfBirth      string        `bson:"dateOfBirth"               json:"dateOfBirth"` // ISO yyyy-mm-dd
	PlaceOfBirth     string        `bson:"placeOfBirth"              json:"placeOfBirth"`
	PlaceOfResidence string        `bson:"placeOfResidence"          json:"placeOfResidence"`
	DocumentType     string        `bson:"documentType"              json:"documentType"`
	DocumentNumber   string        `bson:"documentNumber"            json:"documentNumber"`
	Status           string        `bson:"status"                    json:"status"`
	RejectionReason  string        `bson:"rejectionReason,omitempty" json:"rejectionReason,omitempty"`
	ReviewedBy       string        `bson:"reviewedBy,omitempty"      json:"reviewedBy,omitempty"`
	ReviewedAt       *time.Time    `bson:"reviewedAt,omitempty"      json:"reviewedAt,omitempty"`
	CreatedAt        time.Time     `bson:"createdAt"                 json:"createdAt"`
	UpdatedAt        time.Time     `bson:"updatedAt"                 json:"updatedAt"`
}

// SubmitInput is the customer body for POST /api/v1/kyc.
type SubmitInput struct {
	FullName         string `json:"fullName"`
	DateOfBirth      string `json:"dateOfBirth"`
	PlaceOfBirth     string `json:"placeOfBirth"`
	PlaceOfResidence string `json:"placeOfResidence"`
	DocumentType     string `json:"documentType"`
	DocumentNumber   string `json:"documentNumber"`
}

// Profile is the derived KYC view returned to the customer (GET /api/v1/kyc/me).
// Status is one of unverified|pending|verified|rejected; Submission echoes the
// stored record (so the form can pre-fill on re-submit) when one exists.
type Profile struct {
	Status          string      `json:"status"`
	RejectionReason string      `json:"rejectionReason,omitempty"`
	Submission      *Submission `json:"submission,omitempty"`
}

// profileFor maps a submission (or nil) to the derived customer profile.
func profileFor(s *Submission) *Profile {
	if s == nil {
		return &Profile{Status: ProfileUnverified}
	}
	status := ProfilePending
	switch s.Status {
	case StatusApproved:
		status = ProfileVerified
	case StatusRejected:
		status = ProfileRejected
	}
	return &Profile{Status: status, RejectionReason: s.RejectionReason, Submission: s}
}
