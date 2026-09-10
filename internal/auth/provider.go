package auth

import "golang.org/x/oauth2"

// ProviderConfig is the OAuth2 shape for one mail provider. ClientID comes
// from wherever sift's own app registration is configured; it isn't
// hardcoded here.
type ProviderConfig struct {
	ClientID string
	Endpoint oauth2.Endpoint
	Scopes   []string

	// RedirectHost is the loopback hostname used for the redirect URI.
	// Google's and Microsoft's native-app loopback conventions disagree
	// here: Google requires the literal IP "127.0.0.1" (and rejects
	// "localhost"); Microsoft's public-client wildcard-port redirect URI
	// is registered as exactly "http://localhost" and only matches
	// "localhost" at runtime (and only with no path - see Authenticate).
	RedirectHost string
}

func Gmail(clientID string) ProviderConfig {
	return ProviderConfig{
		ClientID:     clientID,
		RedirectHost: "127.0.0.1",
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.google.com/o/oauth2/v2/auth",
			TokenURL: "https://oauth2.googleapis.com/token",
		},
		Scopes: []string{
			"https://www.googleapis.com/auth/gmail.modify",
			"https://www.googleapis.com/auth/gmail.send",
		},
	}
}

func Outlook(clientID string) ProviderConfig {
	return ProviderConfig{
		ClientID:     clientID,
		RedirectHost: "localhost",
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://login.microsoftonline.com/common/oauth2/v2.0/authorize",
			TokenURL: "https://login.microsoftonline.com/common/oauth2/v2.0/token",
		},
		Scopes: []string{
			"offline_access",
			"https://graph.microsoft.com/Mail.ReadWrite",
			"https://graph.microsoft.com/Mail.Send",
		},
	}
}
