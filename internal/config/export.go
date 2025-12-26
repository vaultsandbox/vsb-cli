package config

import "time"

// ExportedInboxFile is the file format for exported inboxes
type ExportedInboxFile struct {
	Version      int          `json:"version"`
	EmailAddress string       `json:"emailAddress"`
	InboxHash    string       `json:"inboxHash"`
	ExpiresAt    time.Time    `json:"expiresAt"`
	ExportedAt   time.Time    `json:"exportedAt"`
	Keys         ExportedKeys `json:"keys"`
}

type ExportedKeys struct {
	KEMPrivate  string `json:"kemPrivate"`
	KEMPublic   string `json:"kemPublic"`
	ServerSigPK string `json:"serverSigPk"`
}
