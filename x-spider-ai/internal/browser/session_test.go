package browser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMultiAccountSessions(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "xspiderai_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test_sessions.db")
	sm := NewSessionManager(dbPath)

	// Save account 1
	sess1 := &SessionData{
		AccountID:  "main",
		ScreenName: "_ttomsin",
		AuthToken:  "token_main_123",
		CT0:        "ct0_main_456",
	}
	if err := sm.SaveSession(sess1); err != nil {
		t.Fatalf("failed to save session 1: %v", err)
	}

	// Save account 2
	sess2 := &SessionData{
		AccountID:  "bot_alpha",
		ScreenName: "alpha_bot",
		AuthToken:  "token_bot_789",
		CT0:        "ct0_bot_012",
	}
	if err := sm.SaveSession(sess2); err != nil {
		t.Fatalf("failed to save session 2: %v", err)
	}

	// List accounts
	accounts, err := sm.ListAccounts()
	if err != nil {
		t.Fatalf("failed to list accounts: %v", err)
	}
	if len(accounts) != 2 {
		t.Fatalf("expected 2 accounts, got %d", len(accounts))
	}

	// Load specific account by ID
	loaded1, err := sm.LoadAccountSession("main")
	if err != nil {
		t.Fatalf("failed to load main account: %v", err)
	}
	if loaded1.AuthToken != "token_main_123" {
		t.Fatalf("expected token_main_123, got %s", loaded1.AuthToken)
	}

	loaded2, err := sm.LoadAccountSession("bot_alpha")
	if err != nil {
		t.Fatalf("failed to load bot_alpha: %v", err)
	}
	if loaded2.AuthToken != "token_bot_789" {
		t.Fatalf("expected token_bot_789, got %s", loaded2.AuthToken)
	}

	// Switch active account to main
	if err := sm.SwitchAccount("main"); err != nil {
		t.Fatalf("failed to switch account: %v", err)
	}

	active, err := sm.LoadSession()
	if err != nil {
		t.Fatalf("failed to load active session: %v", err)
	}
	if active.AccountID != "main" {
		t.Fatalf("expected active account to be main, got %s", active.AccountID)
	}

	// Delete account 2
	if err := sm.DeleteAccount("bot_alpha"); err != nil {
		t.Fatalf("failed to delete bot_alpha: %v", err)
	}
	accountsAfter, _ := sm.ListAccounts()
	if len(accountsAfter) != 1 {
		t.Fatalf("expected 1 account after delete, got %d", len(accountsAfter))
	}
}
