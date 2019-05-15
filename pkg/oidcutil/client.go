package oidcutil

import (
	"context"
	"errors"
	"fmt"
	"github.com/coreos/go-oidc"
	"golang.org/x/oauth2"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/phayes/freeport"
	"github.com/skratchdot/open-golang/open"
)

var randSource *rand.Rand
var alphabet = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func init(){
	randSource = rand.New(rand.NewSource(time.Now().UnixNano()))
}

func newState() string{
	b := make([]rune, 64)
	for i := range b {
		b[i] = alphabet[randSource.Intn(len(alphabet))]
	}
	return string(b)
}

type Client struct {
	provider *oidc.Provider
	verifier *oidc.IDTokenVerifier
	verifiedEmailClaimCheck bool

	server *http.Server
	oauth2Config *oauth2.Config
	state string

	idTokenSource *IDTokenSource
	callbackWaitCh chan struct{}

	sigCh chan os.Signal
}

func NewClient(issuerURL, clientID, clientSecret string, verifiedEmailClaimCheck bool) (*Client, error) {
	log.Print("DEBUG: start oidcutil.NewClient")
	provider, err := oidc.NewProvider(context.Background(), issuerURL)

	if err != nil {
		return nil, err
	}

	idTokenVerifier := provider.Verifier(&oidc.Config{ClientID: clientID})

	log.Print("DEBUG: finished oidc provider/initialization")

	port, err := freeport.GetFreePort()
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	listen := fmt.Sprintf("localhost:%d", port)
	log.Printf("INFO: Client is listening %s", listen)
	mux := http.NewServeMux()
	srv := &http.Server{Addr: listen, Handler: mux}
	oauth2Config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  fmt.Sprintf("http://localhost:%d/callback", port),
		Endpoint:     provider.Endpoint(),
		Scopes:       []string{
			oidc.ScopeOpenID,
			oidc.ScopeOfflineAccess,
			"email",
		},
	}
	c := &Client{
		provider:       provider,
		verifier:       idTokenVerifier,
		verifiedEmailClaimCheck:  verifiedEmailClaimCheck,
		server: srv,
		oauth2Config:   oauth2Config,
		callbackWaitCh: make(chan struct{}),
	}
	mux.HandleFunc("/callback", c.handleCallback)
	go func() {
		if err := c.server.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()

	log.Print("DEBUG: finish oidcutil.NewClient")
	return c, nil
}

func (c *Client) Shutdown(ctx context.Context) error {
	_ctx, _ := context.WithTimeout(ctx, 5*time.Second)
	if err := c.server.Shutdown(_ctx); err != nil {
		log.Fatal(err)
		return err
	}
	return nil
}

func (c *Client) handleCallback(w http.ResponseWriter, r *http.Request) {
	log.Print("DEBUG: start oidcutil.Client.handleCallback")

	err := c.exchange(r)
	if err != nil {
		log.Printf("ERROR: %s", err)
		w.WriteHeader(500)
		w.Write([]byte(`{
  "status": "error",
  "message": "please close this page."
}`))
	}

	w.WriteHeader(200)
	w.Write([]byte(`{
  "status": "succeeded",
  "message": "please close this page."
}`))
	c.callbackWaitCh <- struct {}{}
	log.Print("DEBUG: finish oidcutil.Client.handleCallback")
}

func (c *Client) exchange(r *http.Request) error {
	log.Print("DEBUG: start oidcutil.Client.exchange")

	// nonce check
	state := r.URL.Query().Get("state")
	if c.state != state {
		return fmt.Errorf("state parameter mismatch: expected=%s, actual=%s", c.state, state)
	}

	oauth2Token, err := c.oauth2Config.Exchange(context.Background(), r.URL.Query().Get("code"))
	if err != nil {
		return err
	}

	log.Printf("DEBUG: received new oauth2 token: %+v", oauth2Token)

	c.idTokenSource = NewIDTokenSource(
		c.verifier,
		c.oauth2Config.TokenSource(context.Background(), oauth2Token),
		c.verifiedEmailClaimCheck,
	)

	log.Print("DEBUG: finish oidcutil.Client.exchange")
	return nil
}

func (c *Client) authURL() string {
	return c.oauth2Config.AuthCodeURL(c.state)
}

func (c* Client) token() (*TokenWrapper, error) {
	if c.idTokenSource == nil {
		return nil, errors.New("couldn't retrieve verified new id token")
	}
	t, err := c.idTokenSource.Token()
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (c *Client) retrieveNewToken(ctx context.Context) (*TokenWrapper, error) {
	log.Print("DEBUG: start oidcutil.Client.retrieveNewToken")

	c.idTokenSource = nil
	c.state = newState()
	authURL := c.authURL()

	log.Print("INFO: retrieving vew id token")
	log.Print("INFO: opening ", authURL)
	if err := open.Start(authURL); err != nil {
		return nil, err
	}

	var t *TokenWrapper
	var err error
	L: for {
		select {
		case <-ctx.Done():
			err = ctx.Err()
			break L
		case <-c.callbackWaitCh:
			t, err =  c.token()
			break L
		}
	}

	log.Print("DEBUG: finish oidcutil.Client.retrieveNewToken")
	return t, err
}

func (c *Client) Authenticate(ctx context.Context) (*TokenWrapper, error) {
	log.Print("DEBUG: start oidcutil.Client.Authenticate")

	t, err := c.token()

	if err == nil {
		return t, nil
	}

	// needs retrieve new token forever (or canceled)
	L: for err != nil {
		log.Print("INFO: couldn't fetch id token: ", err)
		log.Print("INFO: retrieving new id token again.")
		select {
		case <-ctx.Done():
			err = ctx.Err()
			break L
		default:
			childCtx, _ := context.WithCancel(ctx)
			t, err = c.retrieveNewToken(childCtx)
			break L
		}
	}

	return t, err
}

