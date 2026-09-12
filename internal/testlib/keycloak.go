//go:build integration

package testlib

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/waypoint-group/waypoint-be/internal/api/middleware"
)

//go:embed testdata/realm.json
var keycloakRealm []byte

type Keycloak struct {
	ctr        *testcontainers.DockerContainer
	URL        string
	JWTConfig  middleware.JWTConfig
	cancelKeys context.CancelFunc
}

func NewKeycloak() (*Keycloak, error) {
	parent := context.Background()

	ctr, err := testcontainers.Run(
		parent,
		"quay.io/keycloak/keycloak:26.7.3",
		testcontainers.WithCmd("start-dev", "--import-realm"),
		testcontainers.WithExposedPorts("8080/tcp"),
		testcontainers.WithFiles(testcontainers.ContainerFile{
			ContainerFilePath: "/opt/keycloak/data/import/waypoint-realm.json",
			Reader:            bytes.NewReader(keycloakRealm),
			FileMode:          0o644, // rw-r--r--
		}),
		testcontainers.WithWaitStrategy(
			wait.ForHTTP("/realms/waypoint/.well-known/openid-configuration").
				WithPort("8080/tcp").
				WithStartupTimeout(time.Minute),
		),
	)
	kc := &Keycloak{ctr: ctr}
	if err != nil {
		kc.Close()
		return nil, fmt.Errorf("failed to start Keycloak: %w", err)
	}

	kc.URL, err = ctr.PortEndpoint(parent, "8080/tcp", "http")
	if err != nil {
		kc.Close()
		return nil, fmt.Errorf("failed to get Keycloak URL: %w", err)
	}

	issuerURL := kc.URL + "/realms/waypoint"
	keysCtx, cancelKeys := context.WithCancel(parent)
	kc.cancelKeys = cancelKeys
	keys, err := keyfunc.NewDefaultCtx(keysCtx, []string{issuerURL + "/protocol/openid-connect/certs"})
	if err != nil {
		kc.Close()
		return nil, fmt.Errorf("create Keycloak key resolver: %w", err)
	}
	kc.JWTConfig = middleware.JWTConfig{
		Issuer:   issuerURL,
		Audience: "waypoint-api",
		KeyFunc:  keys.Keyfunc,
	}
	return kc, nil
}

// AccessToken logs a fixture user into the test-only direct grant client.
func (kc *Keycloak) AccessToken(
	ctx context.Context,
	username,
	password string,
) (accessToken string, accessTokenErr error) {
	form := url.Values{
		"grant_type": {"password"},
		"client_id":  {"waypoint-tests"},
		"username":   {username},
		"password":   {password},
	}
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		kc.JWTConfig.Issuer+"/protocol/openid-connect/token",
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return "", fmt.Errorf("create token request: %w", err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 10 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		return "", fmt.Errorf("request Keycloak token: %w", err)
	}

	defer func() {
		if err := response.Body.Close(); err != nil {
			accessTokenErr = errors.Join(
				err,
				accessTokenErr,
			)
		}
	}()

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Keycloak token request returned %s", response.Status)
	}

	var result struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode Keycloak token: %w", err)
	}
	if result.AccessToken == "" {
		return "", fmt.Errorf("Keycloak returned an empty access token")
	}

	return result.AccessToken, nil
}

func (kc *Keycloak) Close() {
	if kc.cancelKeys != nil {
		kc.cancelKeys()
	}
	if kc.ctr == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_ = kc.ctr.Terminate(ctx)
}
