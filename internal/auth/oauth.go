package auth

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"runtime"

	"golang.org/x/oauth2"
)

// Authenticate runs a loopback OAuth2 + PKCE flow: it starts a local
// callback listener, hands the caller the URL to open (openBrowser is
// injectable so tests don't need a real browser), and blocks until the
// provider redirects back with a code or the context is done.
func Authenticate(ctx context.Context, cfg ProviderConfig, openBrowser func(url string) error) (tok *oauth2.Token, err error) {
	verifier, challenge, err := newPKCE()
	if err != nil {
		return nil, err
	}
	state, err := randomState()
	if err != nil {
		return nil, err
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("start callback listener: %w", err)
	}

	oconf := &oauth2.Config{
		ClientID:    cfg.ClientID,
		Endpoint:    cfg.Endpoint,
		Scopes:      cfg.Scopes,
		RedirectURL: fmt.Sprintf("http://127.0.0.1:%d/callback", ln.Addr().(*net.TCPAddr).Port),
	}

	type callbackResult struct {
		code     string
		err      error
		writeErr error
	}
	results := make(chan callbackResult, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		switch {
		case q.Get("error") != "":
			results <- callbackResult{err: fmt.Errorf("authorization denied: %s", q.Get("error"))}
			http.Error(w, "Authorization failed. You can close this window.", http.StatusOK)
		case q.Get("state") != state:
			results <- callbackResult{err: errors.New("state mismatch")}
			http.Error(w, "Authorization failed: state mismatch.", http.StatusBadRequest)
		default:
			_, writeErr := fmt.Fprint(w, "Authorization complete. You can close this window and return to sift.")
			results <- callbackResult{code: q.Get("code"), writeErr: writeErr}
		}
	})

	srv := &http.Server{Handler: mux}
	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.Serve(ln) }()
	defer func() {
		err = errors.Join(err, srv.Close())
		if se := <-serveErr; se != nil && !errors.Is(se, http.ErrServerClosed) {
			err = errors.Join(err, se)
		}
	}()

	authURL := oconf.AuthCodeURL(state,
		oauth2.SetAuthURLParam("code_challenge", challenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"))
	if err := openBrowser(authURL); err != nil {
		return nil, fmt.Errorf("open browser: %w", err)
	}

	select {
	case res := <-results:
		if res.err != nil {
			return nil, errors.Join(res.err, res.writeErr)
		}
		tok, err = oconf.Exchange(ctx, res.code, oauth2.SetAuthURLParam("code_verifier", verifier))
		return tok, errors.Join(err, res.writeErr)
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// OpenBrowser opens url in the system's default browser.
func OpenBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}
