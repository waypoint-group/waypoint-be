//go:build integration

package integration

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"uuid"

	userapi "github.com/waypoint-group/waypoint-be/internal/user"
	workspaceapi "github.com/waypoint-group/waypoint-be/internal/workspace"
	"github.com/waypoint-group/waypoint-be/tests/testlib"
)

func TestWorkspaces_CreateMissingAuthorization(t *testing.T) {
	uut := newWaypoint(t)
	createTestUserChecked(t, uut, "ada@example.com", "ada", "Ada")
	token := workspaceToken(t, "ada")
	created := createTestWorkspace(t, uut, token, "Engineering")

	response, err := uut.Request(t.Context(), http.MethodPost, "/workspaces", strings.NewReader("{}"))
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusUnauthorized {
		payload, _ := io.ReadAll(response.Body)
		t.Fatalf("expected status %d, got %d: %s", http.StatusUnauthorized, response.StatusCode, payload)
	}
	assertWorkspaceMemberships(t, uut, token, created)
}

func TestWorkspaces_CreateInvalidToken(t *testing.T) {
	uut := newWaypoint(t)
	createTestUserChecked(t, uut, "ada@example.com", "ada", "Ada")
	token := workspaceToken(t, "ada")
	created := createTestWorkspace(t, uut, token, "Engineering")

	response, err := uut.RequestWithAccessToken(t.Context(), http.MethodPost, "/workspaces", strings.NewReader("{}"), "invalid-token")
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusUnauthorized {
		payload, _ := io.ReadAll(response.Body)
		t.Fatalf("expected status %d, got %d: %s", http.StatusUnauthorized, response.StatusCode, payload)
	}
	assertWorkspaceMemberships(t, uut, token, created)
}

func TestWorkspaces_CreateUnregisteredIdentity(t *testing.T) {
	uut := newWaypoint(t)
	createTestUserChecked(t, uut, "ada@example.com", "ada", "Ada")
	token := workspaceToken(t, "ada")
	created := createTestWorkspace(t, uut, token, "Engineering")

	response, err := uut.RequestWithAccessToken(t.Context(), http.MethodPost, "/workspaces", strings.NewReader("{}"), workspaceToken(t, "missing"))
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	defer func() { _ = response.Body.Close() }()

	payload, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	if response.StatusCode != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d: %s", http.StatusForbidden, response.StatusCode, payload)
	}
	if got := strings.TrimSpace(string(payload)); got != userapi.ErrForbidden.Error() {
		t.Fatalf("expected error %q, got %q", userapi.ErrForbidden, got)
	}
	assertWorkspaceMemberships(t, uut, token, created)
}

func TestWorkspaces_CreateMissingName(t *testing.T) {
	uut := newWaypoint(t)
	createTestUserChecked(t, uut, "ada@example.com", "ada", "Ada")
	token := workspaceToken(t, "ada")
	created := createTestWorkspace(t, uut, token, "Engineering")

	response, err := uut.RequestWithAccessToken(t.Context(), http.MethodPost, "/workspaces", strings.NewReader(`{}`), token)
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusBadRequest {
		payload, _ := io.ReadAll(response.Body)
		t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, response.StatusCode, payload)
	}
	assertWorkspaceMemberships(t, uut, token, created)
}

func TestWorkspaces_CreateBlankName(t *testing.T) {
	uut := newWaypoint(t)
	createTestUserChecked(t, uut, "ada@example.com", "ada", "Ada")
	token := workspaceToken(t, "ada")
	created := createTestWorkspace(t, uut, token, "Engineering")

	response, err := uut.RequestWithAccessToken(t.Context(), http.MethodPost, "/workspaces", strings.NewReader(`{"name":"  "}`), token)
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusBadRequest {
		payload, _ := io.ReadAll(response.Body)
		t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, response.StatusCode, payload)
	}
	assertWorkspaceMemberships(t, uut, token, created)
}

func TestWorkspaces_CreateMalformedJSON(t *testing.T) {
	uut := newWaypoint(t)
	createTestUserChecked(t, uut, "ada@example.com", "ada", "Ada")
	token := workspaceToken(t, "ada")
	created := createTestWorkspace(t, uut, token, "Engineering")

	response, err := uut.RequestWithAccessToken(t.Context(), http.MethodPost, "/workspaces", strings.NewReader(`{`), token)
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusBadRequest {
		payload, _ := io.ReadAll(response.Body)
		t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, response.StatusCode, payload)
	}
	assertWorkspaceMemberships(t, uut, token, created)
}

func TestWorkspaces_GetInvalidID(t *testing.T) {
	uut := newWaypoint(t)
	createTestUserChecked(t, uut, "ada@example.com", "ada", "Ada")
	token := workspaceToken(t, "ada")
	created := createTestWorkspace(t, uut, token, "Engineering")

	response, err := uut.RequestWithAccessToken(t.Context(), http.MethodGet, "/workspaces/"+"invalid", nil, token)
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	defer func() { _ = response.Body.Close() }()

	payload, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, response.StatusCode, payload)
	}
	if got := strings.TrimSpace(string(payload)); got != workspaceapi.ErrInvalidID.Error() {
		t.Fatalf("expected error %q, got %q", workspaceapi.ErrInvalidID, got)
	}
	assertWorkspaceMemberships(t, uut, token, created)
}

func TestWorkspaces_DeleteInvalidID(t *testing.T) {
	uut := newWaypoint(t)
	createTestUserChecked(t, uut, "ada@example.com", "ada", "Ada")
	token := workspaceToken(t, "ada")
	created := createTestWorkspace(t, uut, token, "Engineering")

	response, err := uut.RequestWithAccessToken(t.Context(), http.MethodDelete, "/workspaces/"+"invalid", nil, token)
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	defer func() { _ = response.Body.Close() }()

	payload, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, response.StatusCode, payload)
	}
	if got := strings.TrimSpace(string(payload)); got != workspaceapi.ErrInvalidID.Error() {
		t.Fatalf("expected error %q, got %q", workspaceapi.ErrInvalidID, got)
	}
	assertWorkspaceMemberships(t, uut, token, created)
}

func TestWorkspaces_AddMemberInvalidUserID(t *testing.T) {
	uut := newWaypoint(t)
	createTestUserChecked(t, uut, "ada@example.com", "ada", "Ada")
	token := workspaceToken(t, "ada")
	created := createTestWorkspace(t, uut, token, "Engineering")
	path := "/workspaces/" + created.ID.String()

	response, err := uut.RequestWithAccessToken(t.Context(), http.MethodPost, path+"/users/"+"invalid", nil, token)
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	defer func() { _ = response.Body.Close() }()

	payload, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, response.StatusCode, payload)
	}
	if got := strings.TrimSpace(string(payload)); got != workspaceapi.ErrInvalidID.Error() {
		t.Fatalf("expected error %q, got %q", workspaceapi.ErrInvalidID, got)
	}
	assertWorkspaceMemberships(t, uut, token, created)
}

func TestWorkspaces_RemoveMemberInvalidUserID(t *testing.T) {
	uut := newWaypoint(t)
	createTestUserChecked(t, uut, "ada@example.com", "ada", "Ada")
	token := workspaceToken(t, "ada")
	created := createTestWorkspace(t, uut, token, "Engineering")
	path := "/workspaces/" + created.ID.String()

	response, err := uut.RequestWithAccessToken(t.Context(), http.MethodDelete, path+"/users/"+"invalid", nil, token)
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	defer func() { _ = response.Body.Close() }()

	payload, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, response.StatusCode, payload)
	}
	if got := strings.TrimSpace(string(payload)); got != workspaceapi.ErrInvalidID.Error() {
		t.Fatalf("expected error %q, got %q", workspaceapi.ErrInvalidID, got)
	}
	assertWorkspaceMemberships(t, uut, token, created)
}

func TestWorkspaces_AddMemberMissingUser(t *testing.T) {
	uut := newWaypoint(t)
	createTestUserChecked(t, uut, "ada@example.com", "ada", "Ada")
	token := workspaceToken(t, "ada")
	created := createTestWorkspace(t, uut, token, "Engineering")
	path := "/workspaces/" + created.ID.String()

	response, err := uut.RequestWithAccessToken(t.Context(), http.MethodPost, path+"/users/"+uuid.NewV7().String(), nil, token)
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	defer func() { _ = response.Body.Close() }()

	payload, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d: %s", http.StatusNotFound, response.StatusCode, payload)
	}
	if got := strings.TrimSpace(string(payload)); got != workspaceapi.ErrUserNotFound.Error() {
		t.Fatalf("expected error %q, got %q", workspaceapi.ErrUserNotFound, got)
	}
	assertWorkspaceMemberships(t, uut, token, created)
}

func TestWorkspaces_RemoveMemberMissingUser(t *testing.T) {
	uut := newWaypoint(t)
	createTestUserChecked(t, uut, "ada@example.com", "ada", "Ada")
	token := workspaceToken(t, "ada")
	created := createTestWorkspace(t, uut, token, "Engineering")
	path := "/workspaces/" + created.ID.String()

	response, err := uut.RequestWithAccessToken(t.Context(), http.MethodDelete, path+"/users/"+uuid.NewV7().String(), nil, token)
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	defer func() { _ = response.Body.Close() }()

	payload, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d: %s", http.StatusNotFound, response.StatusCode, payload)
	}
	if got := strings.TrimSpace(string(payload)); got != workspaceapi.ErrWorkspaceMemberNotFound.Error() {
		t.Fatalf("expected error %q, got %q", workspaceapi.ErrWorkspaceMemberNotFound, got)
	}
	assertWorkspaceMemberships(t, uut, token, created)
}

func TestWorkspaces_Create(t *testing.T) {
	uut := newWaypoint(t)
	owner := createTestUserChecked(t, uut, "ada@example.com", "ada", "Ada")
	token := workspaceToken(t, "ada")
	assertWorkspaceMemberships(t, uut, token)
	created := createTestWorkspace(t, uut, token, "  Engineering  ")

	if created.Name != "Engineering" || created.ID == (uuid.UUID{}) || created.CreatedAt.IsZero() {
		t.Fatalf("unexpected workspace: %+v", created)
	}
	if len(created.Members) != 1 || created.Members[0].UserID.String() != owner.ID || created.Members[0].Role != workspaceapi.WorkspaceRoleOwner {
		t.Fatalf("expected creator as sole owner, got %+v", created.Members)
	}

	assertWorkspaceMemberships(t, uut, token, created)
}

func TestWorkspaces_GetNonMember(t *testing.T) {
	uut := newWaypoint(t)
	createTestUserChecked(t, uut, "ada@example.com", "ada", "Ada")
	token := workspaceToken(t, "ada")
	created := createTestWorkspace(t, uut, token, "Engineering")
	path := "/workspaces/" + created.ID.String()
	createTestUserChecked(t, uut, "linked-ada@example.com", "linked-ada", "Another Ada")
	memberToken := workspaceToken(t, "linked-ada")

	response, err := uut.RequestWithAccessToken(t.Context(), http.MethodGet, path, nil, memberToken)
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	defer func() { _ = response.Body.Close() }()

	payload, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	if response.StatusCode != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d: %s", http.StatusForbidden, response.StatusCode, payload)
	}
	if got := strings.TrimSpace(string(payload)); got != workspaceapi.ErrForbidden.Error() {
		t.Fatalf("expected error %q, got %q", workspaceapi.ErrForbidden, got)
	}
	assertWorkspaceMemberships(t, uut, memberToken)
}

func TestWorkspaces_AddMember(t *testing.T) {
	uut := newWaypoint(t)
	createTestUserChecked(t, uut, "ada@example.com", "ada", "Ada")
	token := workspaceToken(t, "ada")
	created := createTestWorkspace(t, uut, token, "Engineering")
	path := "/workspaces/" + created.ID.String()
	member := createTestUserChecked(t, uut, "linked-ada@example.com", "linked-ada", "Another Ada")
	memberToken := workspaceToken(t, "linked-ada")
	membershipPath := path + "/users/" + member.ID

	response, err := uut.RequestWithAccessToken(t.Context(), http.MethodPost, membershipPath, nil, token)
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusCreated {
		payload, _ := io.ReadAll(response.Body)
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, response.StatusCode, payload)
	}
	if got := response.Header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected JSON content type, got %q", got)
	}
	var added workspaceapi.WorkspaceMember
	if err := json.NewDecoder(response.Body).Decode(&added); err != nil {
		t.Fatal(err)
	}
	if added.UserID.String() != member.ID || added.Role != workspaceapi.WorkspaceRoleMember || added.WorkspaceID != created.ID {
		t.Fatalf("unexpected membership: %+v", added)
	}

	assertWorkspaceMemberships(t, uut, memberToken, created)
}

func TestWorkspaces_GetMember(t *testing.T) {
	uut := newWaypoint(t)
	createTestUserChecked(t, uut, "ada@example.com", "ada", "Ada")
	token := workspaceToken(t, "ada")
	created := createTestWorkspace(t, uut, token, "Engineering")
	path := "/workspaces/" + created.ID.String()
	member := createTestUserChecked(t, uut, "linked-ada@example.com", "linked-ada", "Another Ada")
	memberToken := workspaceToken(t, "linked-ada")
	membershipPath := path + "/users/" + member.ID
	addMemberResponse, err := uut.RequestWithAccessToken(t.Context(), http.MethodPost, membershipPath, nil, token)
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	defer func() { _ = addMemberResponse.Body.Close() }()

	if addMemberResponse.StatusCode != http.StatusCreated {
		addMemberBody, _ := io.ReadAll(addMemberResponse.Body)
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, addMemberResponse.StatusCode, addMemberBody)
	}
	if got := addMemberResponse.Header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected JSON content type, got %q", got)
	}

	response, err := uut.RequestWithAccessToken(t.Context(), http.MethodGet, path, nil, memberToken)
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusOK {
		payload, _ := io.ReadAll(response.Body)
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, response.StatusCode, payload)
	}
	if got := response.Header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected JSON content type, got %q", got)
	}
	var read workspaceapi.Workspace
	if err := json.NewDecoder(response.Body).Decode(&read); err != nil {
		t.Fatal(err)
	}
	if read.ID != created.ID || read.Name != created.Name || len(read.Members) != 2 {
		t.Fatalf("unexpected workspace: %+v", read)
	}
}

func TestWorkspaces_AddExistingMember(t *testing.T) {
	uut := newWaypoint(t)
	createTestUserChecked(t, uut, "ada@example.com", "ada", "Ada")
	token := workspaceToken(t, "ada")
	created := createTestWorkspace(t, uut, token, "Engineering")
	path := "/workspaces/" + created.ID.String()
	member := createTestUserChecked(t, uut, "linked-ada@example.com", "linked-ada", "Another Ada")
	membershipPath := path + "/users/" + member.ID
	addMemberResponse, err := uut.RequestWithAccessToken(t.Context(), http.MethodPost, membershipPath, nil, token)
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	defer func() { _ = addMemberResponse.Body.Close() }()

	if addMemberResponse.StatusCode != http.StatusCreated {
		addMemberBody, _ := io.ReadAll(addMemberResponse.Body)
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, addMemberResponse.StatusCode, addMemberBody)
	}
	if got := addMemberResponse.Header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected JSON content type, got %q", got)
	}

	response, err := uut.RequestWithAccessToken(t.Context(), http.MethodPost, membershipPath, nil, token)
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	defer func() { _ = response.Body.Close() }()

	payload, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	if response.StatusCode != http.StatusConflict {
		t.Fatalf("expected status %d, got %d: %s", http.StatusConflict, response.StatusCode, payload)
	}
	if got := strings.TrimSpace(string(payload)); got != workspaceapi.ErrAlreadyMember.Error() {
		t.Fatalf("expected error %q, got %q", workspaceapi.ErrAlreadyMember, got)
	}
}

func TestWorkspaces_DeleteAsMember(t *testing.T) {
	uut := newWaypoint(t)
	createTestUserChecked(t, uut, "ada@example.com", "ada", "Ada")
	token := workspaceToken(t, "ada")
	created := createTestWorkspace(t, uut, token, "Engineering")
	path := "/workspaces/" + created.ID.String()
	member := createTestUserChecked(t, uut, "linked-ada@example.com", "linked-ada", "Another Ada")
	memberToken := workspaceToken(t, "linked-ada")
	membershipPath := path + "/users/" + member.ID
	addMemberResponse, err := uut.RequestWithAccessToken(t.Context(), http.MethodPost, membershipPath, nil, token)
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	defer func() { _ = addMemberResponse.Body.Close() }()

	if addMemberResponse.StatusCode != http.StatusCreated {
		addMemberBody, _ := io.ReadAll(addMemberResponse.Body)
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, addMemberResponse.StatusCode, addMemberBody)
	}
	if got := addMemberResponse.Header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected JSON content type, got %q", got)
	}

	response, err := uut.RequestWithAccessToken(t.Context(), http.MethodDelete, path, nil, memberToken)
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	defer func() { _ = response.Body.Close() }()

	payload, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	if response.StatusCode != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d: %s", http.StatusForbidden, response.StatusCode, payload)
	}
	if got := strings.TrimSpace(string(payload)); got != workspaceapi.ErrForbidden.Error() {
		t.Fatalf("expected error %q, got %q", workspaceapi.ErrForbidden, got)
	}
}

func TestWorkspaces_RemoveMemberAsMember(t *testing.T) {
	uut := newWaypoint(t)
	createTestUserChecked(t, uut, "ada@example.com", "ada", "Ada")
	token := workspaceToken(t, "ada")
	created := createTestWorkspace(t, uut, token, "Engineering")
	path := "/workspaces/" + created.ID.String()
	member := createTestUserChecked(t, uut, "linked-ada@example.com", "linked-ada", "Another Ada")
	memberToken := workspaceToken(t, "linked-ada")
	membershipPath := path + "/users/" + member.ID
	addMemberResponse, err := uut.RequestWithAccessToken(t.Context(), http.MethodPost, membershipPath, nil, token)
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	defer func() { _ = addMemberResponse.Body.Close() }()

	if addMemberResponse.StatusCode != http.StatusCreated {
		addMemberBody, _ := io.ReadAll(addMemberResponse.Body)
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, addMemberResponse.StatusCode, addMemberBody)
	}
	if got := addMemberResponse.Header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected JSON content type, got %q", got)
	}

	response, err := uut.RequestWithAccessToken(t.Context(), http.MethodDelete, membershipPath, nil, memberToken)
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	defer func() { _ = response.Body.Close() }()

	payload, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	if response.StatusCode != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d: %s", http.StatusForbidden, response.StatusCode, payload)
	}
	if got := strings.TrimSpace(string(payload)); got != workspaceapi.ErrForbidden.Error() {
		t.Fatalf("expected error %q, got %q", workspaceapi.ErrForbidden, got)
	}
}

func TestWorkspaces_AddMemberAsMember(t *testing.T) {
	uut := newWaypoint(t)
	owner := createTestUserChecked(t, uut, "ada@example.com", "ada", "Ada")
	token := workspaceToken(t, "ada")
	created := createTestWorkspace(t, uut, token, "Engineering")
	path := "/workspaces/" + created.ID.String()
	member := createTestUserChecked(t, uut, "linked-ada@example.com", "linked-ada", "Another Ada")
	memberToken := workspaceToken(t, "linked-ada")
	membershipPath := path + "/users/" + member.ID
	addMemberResponse, err := uut.RequestWithAccessToken(t.Context(), http.MethodPost, membershipPath, nil, token)
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	defer func() { _ = addMemberResponse.Body.Close() }()

	if addMemberResponse.StatusCode != http.StatusCreated {
		addMemberBody, _ := io.ReadAll(addMemberResponse.Body)
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, addMemberResponse.StatusCode, addMemberBody)
	}
	if got := addMemberResponse.Header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected JSON content type, got %q", got)
	}

	response, err := uut.RequestWithAccessToken(t.Context(), http.MethodPost, path+"/users/"+owner.ID, nil, memberToken)
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	defer func() { _ = response.Body.Close() }()

	payload, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	if response.StatusCode != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d: %s", http.StatusForbidden, response.StatusCode, payload)
	}
	if got := strings.TrimSpace(string(payload)); got != workspaceapi.ErrForbidden.Error() {
		t.Fatalf("expected error %q, got %q", workspaceapi.ErrForbidden, got)
	}
}

func TestWorkspaces_RemoveOwner(t *testing.T) {
	uut := newWaypoint(t)
	owner := createTestUserChecked(t, uut, "ada@example.com", "ada", "Ada")
	token := workspaceToken(t, "ada")
	created := createTestWorkspace(t, uut, token, "Engineering")
	path := "/workspaces/" + created.ID.String()
	member := createTestUserChecked(t, uut, "linked-ada@example.com", "linked-ada", "Another Ada")
	membershipPath := path + "/users/" + member.ID
	addMemberResponse, err := uut.RequestWithAccessToken(t.Context(), http.MethodPost, membershipPath, nil, token)
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	defer func() { _ = addMemberResponse.Body.Close() }()

	if addMemberResponse.StatusCode != http.StatusCreated {
		addMemberBody, _ := io.ReadAll(addMemberResponse.Body)
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, addMemberResponse.StatusCode, addMemberBody)
	}
	if got := addMemberResponse.Header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected JSON content type, got %q", got)
	}

	response, err := uut.RequestWithAccessToken(t.Context(), http.MethodDelete, path+"/users/"+owner.ID, nil, token)
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	defer func() { _ = response.Body.Close() }()

	payload, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	if response.StatusCode != http.StatusConflict {
		t.Fatalf("expected status %d, got %d: %s", http.StatusConflict, response.StatusCode, payload)
	}
	if got := strings.TrimSpace(string(payload)); got != workspaceapi.ErrOwnerRemoval.Error() {
		t.Fatalf("expected error %q, got %q", workspaceapi.ErrOwnerRemoval, got)
	}
}

func TestWorkspaces_RemoveMember(t *testing.T) {
	uut := newWaypoint(t)
	createTestUserChecked(t, uut, "ada@example.com", "ada", "Ada")
	token := workspaceToken(t, "ada")
	created := createTestWorkspace(t, uut, token, "Engineering")
	path := "/workspaces/" + created.ID.String()
	member := createTestUserChecked(t, uut, "linked-ada@example.com", "linked-ada", "Another Ada")
	memberToken := workspaceToken(t, "linked-ada")
	membershipPath := path + "/users/" + member.ID
	addMemberResponse, err := uut.RequestWithAccessToken(t.Context(), http.MethodPost, membershipPath, nil, token)
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	defer func() { _ = addMemberResponse.Body.Close() }()

	if addMemberResponse.StatusCode != http.StatusCreated {
		addMemberBody, _ := io.ReadAll(addMemberResponse.Body)
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, addMemberResponse.StatusCode, addMemberBody)
	}
	if got := addMemberResponse.Header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected JSON content type, got %q", got)
	}

	response, err := uut.RequestWithAccessToken(t.Context(), http.MethodDelete, membershipPath, nil, token)
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	defer func() { _ = response.Body.Close() }()

	payload, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	if response.StatusCode != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d: %s", http.StatusNoContent, response.StatusCode, payload)
	}
	if len(payload) != 0 {
		t.Fatalf("expected empty body, got %s", payload)
	}
	assertWorkspaceMemberships(t, uut, memberToken)
}

func TestWorkspaces_Delete(t *testing.T) {
	uut := newWaypoint(t)
	createTestUserChecked(t, uut, "ada@example.com", "ada", "Ada")
	token := workspaceToken(t, "ada")
	created := createTestWorkspace(t, uut, token, "Engineering")
	path := "/workspaces/" + created.ID.String()

	response, err := uut.RequestWithAccessToken(t.Context(), http.MethodDelete, path, nil, token)
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	defer func() { _ = response.Body.Close() }()

	payload, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	if response.StatusCode != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d: %s", http.StatusNoContent, response.StatusCode, payload)
	}
	if len(payload) != 0 {
		t.Fatalf("expected empty body, got %s", payload)
	}
	assertWorkspaceMemberships(t, uut, token)
}

func TestWorkspaces_GetMissing(t *testing.T) {
	uut := newWaypoint(t)
	createTestUserChecked(t, uut, "ada@example.com", "ada", "Ada")
	token := workspaceToken(t, "ada")
	path := "/workspaces/" + uuid.NewV7().String()

	response, err := uut.RequestWithAccessToken(t.Context(), http.MethodGet, path, nil, token)
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	defer func() { _ = response.Body.Close() }()

	payload, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d: %s", http.StatusNotFound, response.StatusCode, payload)
	}
	if got := strings.TrimSpace(string(payload)); got != workspaceapi.ErrWorkspaceNotFound.Error() {
		t.Fatalf("expected error %q, got %q", workspaceapi.ErrWorkspaceNotFound, got)
	}
}

func TestWorkspaces_DeleteMissing(t *testing.T) {
	uut := newWaypoint(t)
	createTestUserChecked(t, uut, "ada@example.com", "ada", "Ada")
	token := workspaceToken(t, "ada")
	path := "/workspaces/" + uuid.NewV7().String()

	response, err := uut.RequestWithAccessToken(t.Context(), http.MethodDelete, path, nil, token)
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	defer func() { _ = response.Body.Close() }()

	payload, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d: %s", http.StatusNotFound, response.StatusCode, payload)
	}
	if got := strings.TrimSpace(string(payload)); got != workspaceapi.ErrWorkspaceNotFound.Error() {
		t.Fatalf("expected error %q, got %q", workspaceapi.ErrWorkspaceNotFound, got)
	}
}

// workspaceToken obtains an access token for an identity in the test realm.
func workspaceToken(t *testing.T, username string) string {
	t.Helper()
	token, err := testEnv.Keycloak().AccessToken(t.Context(), username, "test-password")
	if err != nil {
		t.Fatal(err)
	}
	return token
}

// createTestWorkspace creates a workspace through the authenticated HTTP API.
func createTestWorkspace(t *testing.T, uut *testlib.TestWaypoint, token, name string) workspaceapi.Workspace {
	t.Helper()

	body, err := json.Marshal(workspaceapi.CreateWorkspaceRequest{Name: name})
	if err != nil {
		t.Fatal(err)
	}

	response, err := uut.RequestWithAccessToken(t.Context(), http.MethodPost, "/workspaces", bytes.NewReader(body), token)
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusCreated {
		payload, _ := io.ReadAll(response.Body)
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, response.StatusCode, payload)
	}
	if got := response.Header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected JSON content type, got %q", got)
	}
	var result workspaceapi.Workspace
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}

	return result
}

// assertWorkspaceMemberships keeps workspace-related /me assertions in this suite.
func assertWorkspaceMemberships(t *testing.T, uut *testlib.TestWaypoint, token string, expected ...workspaceapi.Workspace) {
	t.Helper()

	response, err := uut.RequestWithAccessToken(t.Context(), http.MethodGet, "/me", nil, token)
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusOK {
		payload, _ := io.ReadAll(response.Body)
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, response.StatusCode, payload)
	}
	if got := response.Header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected JSON content type, got %q", got)
	}
	var profile userapi.MeResponse
	if err := json.NewDecoder(response.Body).Decode(&profile); err != nil {
		t.Fatal(err)
	}

	if profile.WorkspaceMemberships == nil || len(profile.WorkspaceMemberships) != len(expected) {
		t.Fatalf("expected %d memberships in a JSON array, got %+v", len(expected), profile.WorkspaceMemberships)
	}

	for i, want := range expected {
		got := profile.WorkspaceMemberships[i]
		if got.WorkspaceID != want.ID.String() || got.WorkspaceName != want.Name {
			t.Fatalf("expected membership %+v, got %+v", want, got)
		}
	}
}
