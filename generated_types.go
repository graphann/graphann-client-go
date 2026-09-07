package graphann

import generated "github.com/graphann/graphann-client-go/generated"

// Type aliases for endpoints newly wired from the spec (backups, batch
// search, compact-all, license status/audit, admin embed-space). These
// endpoints have no legacy hand-written shape to preserve, so their public
// request/response types are the generated ones directly — the SDK just
// re-exports them under the graphann package so callers don't need to
// import graphann/generated themselves.
type (
	// Backups.
	BackupSummary         = generated.BackupSummary
	BackupManifest        = generated.BackupManifest
	BackupChunkInfo       = generated.BackupChunkInfo
	BackupMeta            = generated.BackupMeta
	CreateBackupResponse  = generated.CreateBackupResponse
	ListBackupsResponse   = generated.ListBackupsResponse
	RestoreBackupRequest  = generated.RestoreBackupRequest
	RestoreBackupResponse = generated.RestoreBackupResponse
	DeleteBackupResponse  = generated.DeleteBackupResponse

	// Batch search.
	BatchSearchRequest  = generated.BatchSearchRequest
	BatchSearchResponse = generated.BatchSearchResponse
	BatchSearchResult   = generated.BatchSearchResult

	// Compact-all.
	CompactAllResponse = generated.CompactAllResponse
	CompactAllSkipped  = generated.CompactAllSkipped

	// License status / audit.
	LicenseStatus       = generated.LicenseStatus
	LicenseEntitlements = generated.LicenseEntitlements
	LicenseAuditEvent   = generated.LicenseAuditEvent

	// Admin embed-space observability.
	EmbedSpaceAdminResponse = generated.EmbedSpaceAdminResponse
	EmbedSpaceIndexRow      = generated.EmbedSpaceIndexRow
)
