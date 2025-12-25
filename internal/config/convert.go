package config

import (
	vaultsandbox "github.com/vaultsandbox/client-go"
)

// StoredInboxFromExport converts SDK ExportedInbox to StoredInbox
func StoredInboxFromExport(exp *vaultsandbox.ExportedInbox) StoredInbox {
	return StoredInbox{
		Email:     exp.EmailAddress,
		ID:        exp.InboxHash,
		CreatedAt: exp.ExportedAt,
		ExpiresAt: exp.ExpiresAt,
		Keys: InboxKeys{
			KEMPrivate:  exp.SecretKeyB64,
			KEMPublic:   exp.PublicKeyB64,
			ServerSigPK: exp.ServerSigPk,
		},
	}
}

// ToExportedInbox converts StoredInbox to SDK ExportedInbox for import
func (s *StoredInbox) ToExportedInbox() *vaultsandbox.ExportedInbox {
	return &vaultsandbox.ExportedInbox{
		EmailAddress: s.Email,
		ExpiresAt:    s.ExpiresAt,
		InboxHash:    s.ID,
		ServerSigPk:  s.Keys.ServerSigPK,
		PublicKeyB64: s.Keys.KEMPublic,
		SecretKeyB64: s.Keys.KEMPrivate,
		ExportedAt:   s.CreatedAt,
	}
}
