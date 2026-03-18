package git

import (
	"sync"
	"time"
)

type SyncSummary struct {
	Added            int `json:"added"`
	Updated          int `json:"updated"`
	Removed          int `json:"removed"`
	Created          int `json:"created"`
	Recreated        int `json:"recreated"`
	Pruned           int `json:"pruned"`
	DriftCorrections int `json:"driftCorrections"`
}

type GitOpsStatus struct {
	Enabled               bool        `json:"enabled"`
	State                 string      `json:"state"` // disabled|initializing|syncing|synced|error
	LastSyncAt            *time.Time  `json:"lastSyncAt,omitempty"`
	LastSuccessfulSyncAt  *time.Time  `json:"lastSuccessfulSyncAt,omitempty"`
	LastChangeDetectedAt  *time.Time  `json:"lastChangeDetectedAt,omitempty"`
	LastSyncCommit        string      `json:"lastSyncCommit,omitempty"`
	LastSyncCommitMessage string      `json:"lastSyncCommitMessage,omitempty"`
	LastError             string      `json:"lastError,omitempty"`
	LastSummary           SyncSummary `json:"lastSummary"`
}

var (
	statusMu     sync.RWMutex
	gitOpsStatus = GitOpsStatus{
		Enabled: false,
		State:   "disabled",
	}
)

func InitializeStatus(enabled bool) {
	statusMu.Lock()
	defer statusMu.Unlock()

	gitOpsStatus.Enabled = enabled
	if enabled {
		gitOpsStatus.State = "initializing"
	} else {
		gitOpsStatus.State = "disabled"
	}
	gitOpsStatus.LastError = ""
	gitOpsStatus.LastSummary = SyncSummary{}
}

func MarkSyncStart(commit string, commitMessage string) {
	statusMu.Lock()
	defer statusMu.Unlock()

	now := time.Now().UTC()
	gitOpsStatus.State = "syncing"
	gitOpsStatus.LastSyncAt = &now
	gitOpsStatus.LastSyncCommit = shortCommit(commit)
	gitOpsStatus.LastSyncCommitMessage = commitMessage
	gitOpsStatus.LastError = ""
}

func MarkChangeDetected() {
	statusMu.Lock()
	defer statusMu.Unlock()

	now := time.Now().UTC()
	gitOpsStatus.LastChangeDetectedAt = &now
}

func MarkSyncSuccess(commit string, commitMessage string, summary SyncSummary) {
	statusMu.Lock()
	defer statusMu.Unlock()

	now := time.Now().UTC()
	gitOpsStatus.State = "synced"
	gitOpsStatus.LastSyncAt = &now
	gitOpsStatus.LastSuccessfulSyncAt = &now
	gitOpsStatus.LastSyncCommit = shortCommit(commit)
	gitOpsStatus.LastSyncCommitMessage = commitMessage
	gitOpsStatus.LastSummary = summary
	gitOpsStatus.LastError = ""
}

func MarkSyncError(commit string, commitMessage string, err error) {
	statusMu.Lock()
	defer statusMu.Unlock()

	now := time.Now().UTC()
	gitOpsStatus.State = "error"
	gitOpsStatus.LastSyncAt = &now
	gitOpsStatus.LastSyncCommit = shortCommit(commit)
	gitOpsStatus.LastSyncCommitMessage = commitMessage
	if err != nil {
		gitOpsStatus.LastError = err.Error()
	}
}

func GetStatus() GitOpsStatus {
	statusMu.RLock()
	defer statusMu.RUnlock()
	return gitOpsStatus
}
