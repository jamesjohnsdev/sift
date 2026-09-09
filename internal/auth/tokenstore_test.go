package auth

import (
	"testing"

	"github.com/zalando/go-keyring"
	"golang.org/x/oauth2"

	"github.com/jamesjohnsdev/sift/internal/provider"
)

func TestTokenStoreRoundTrip(t *testing.T) {
	keyring.MockInit()
	account := provider.AccountID("acct-1")

	got, err := LoadToken(account)
	if err != nil {
		t.Fatalf("LoadToken(unset): %v", err)
	}
	if got != nil {
		t.Fatalf("LoadToken(unset) = %+v, want nil", got)
	}

	want := &oauth2.Token{AccessToken: "at", RefreshToken: "rt"}
	if err := SaveToken(account, want); err != nil {
		t.Fatalf("SaveToken: %v", err)
	}

	got, err = LoadToken(account)
	if err != nil {
		t.Fatalf("LoadToken: %v", err)
	}
	if got == nil || got.AccessToken != want.AccessToken || got.RefreshToken != want.RefreshToken {
		t.Fatalf("LoadToken = %+v, want %+v", got, want)
	}

	if err := DeleteToken(account); err != nil {
		t.Fatalf("DeleteToken: %v", err)
	}
	got, err = LoadToken(account)
	if err != nil {
		t.Fatalf("LoadToken(after delete): %v", err)
	}
	if got != nil {
		t.Fatalf("LoadToken(after delete) = %+v, want nil", got)
	}

	if err := DeleteToken(account); err != nil {
		t.Fatalf("DeleteToken(again, already gone): %v", err)
	}
}
