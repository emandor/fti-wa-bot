package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"wa-bot-notif/go-service/internal/wa"
)

type stubReadiness struct {
	ready bool
}

func (s stubReadiness) Ready() bool { return s.ready }

type stubSender struct {
	err     error
	gotJID  string
	gotText string
}

func (s *stubSender) SendText(_ context.Context, jid, text string) error {
	s.gotJID = jid
	s.gotText = text
	return s.err
}

type stubContacts struct {
	items []wa.ContactSummary
	err   error
}

func (s stubContacts) ListContacts(_ context.Context) ([]wa.ContactSummary, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.items, nil
}

type stubMessages struct {
	items []wa.MessageSummary
}

func (s stubMessages) ListMessages(limit int) []wa.MessageSummary {
	if limit <= 0 || len(s.items) == 0 {
		return nil
	}
	if limit > len(s.items) {
		limit = len(s.items)
	}
	out := make([]wa.MessageSummary, limit)
	copy(out, s.items[:limit])
	return out
}

type stubLogStore struct{}

func (s stubLogStore) InsertSend(_ context.Context, _, _ string) error { return nil }
func (s stubLogStore) InsertUnauthorized(_ context.Context, _, _ string, _ map[string][]string) error {
	return nil
}

type statusPayload struct {
	Status string `json:"status"`
}

type sendPayload struct {
	Success bool   `json:"success"`
	SentTo  string `json:"sent_to"`
}

func TestHealthzAlwaysOK(t *testing.T) {
	h := NewHandler(stubReadiness{ready: false}, &stubSender{}, stubContacts{}, stubMessages{}, stubLogStore{}, Config{AuthToken: "token", GroupJID: "1@g.us"})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var body statusPayload
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if body.Status != "ok" {
		t.Fatalf("expected status body ok, got %q", body.Status)
	}
}

func TestReadyzUsesReadinessState(t *testing.T) {
	hNotReady := NewHandler(stubReadiness{ready: false}, &stubSender{}, stubContacts{}, stubMessages{}, stubLogStore{}, Config{AuthToken: "token", GroupJID: "1@g.us"})
	rrNotReady := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	hNotReady.ServeHTTP(rrNotReady, req)

	if rrNotReady.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", rrNotReady.Code)
	}

	hReady := NewHandler(stubReadiness{ready: true}, &stubSender{}, stubContacts{}, stubMessages{}, stubLogStore{}, Config{AuthToken: "token", GroupJID: "1@g.us"})
	rrReady := httptest.NewRecorder()
	hReady.ServeHTTP(rrReady, req)

	if rrReady.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rrReady.Code)
	}
}

func TestSendUnauthorizedReturns401(t *testing.T) {
	h := NewHandler(stubReadiness{ready: true}, &stubSender{}, stubContacts{}, stubMessages{}, stubLogStore{}, Config{AuthToken: "token", GroupJID: "1@g.us"})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/send", bytes.NewBufferString(`{"message":"hello"}`))

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rr.Code)
	}
}

func TestSendSuccess(t *testing.T) {
	sender := &stubSender{}
	h := NewHandler(stubReadiness{ready: true}, sender, stubContacts{}, stubMessages{}, stubLogStore{}, Config{AuthToken: "token", GroupJID: "1@g.us"})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/send", bytes.NewBufferString(`{"message":"hello"}`))
	req.Header.Set("Authorization", "Bearer token")

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var payload sendPayload
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if !payload.Success {
		t.Fatalf("expected success response")
	}

	if payload.SentTo != "1@g.us" {
		t.Fatalf("expected sent_to to use fallback group, got %q", payload.SentTo)
	}

	if sender.gotJID != "1@g.us" {
		t.Fatalf("expected sender target 1@g.us, got %q", sender.gotJID)
	}
}

func TestSendWithUserIDUsesUserJID(t *testing.T) {
	sender := &stubSender{}
	h := NewHandler(stubReadiness{ready: true}, sender, stubContacts{}, stubMessages{}, stubLogStore{}, Config{AuthToken: "token", GroupJID: "1@g.us"})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/send", bytes.NewBufferString(`{"message":"hello","userId":"62812-345 67"}`))
	req.Header.Set("Authorization", "Bearer token")

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	if sender.gotJID != "6281234567@s.whatsapp.net" {
		t.Fatalf("expected normalized user jid, got %q", sender.gotJID)
	}
}

func TestSendWhenNotReadyReturns503(t *testing.T) {
	h := NewHandler(stubReadiness{ready: true}, &stubSender{err: wa.ErrNotReady}, stubContacts{}, stubMessages{}, stubLogStore{}, Config{AuthToken: "token", GroupJID: "1@g.us"})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/send", bytes.NewBufferString(`{"message":"hello"}`))
	req.Header.Set("Authorization", "Bearer token")

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", rr.Code)
	}
}

func TestSendGatewayErrorReturns502(t *testing.T) {
	h := NewHandler(stubReadiness{ready: true}, &stubSender{err: errors.New("send failed")}, stubContacts{}, stubMessages{}, stubLogStore{}, Config{AuthToken: "token", GroupJID: "1@g.us"})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/send", bytes.NewBufferString(`{"message":"hello"}`))
	req.Header.Set("Authorization", "Bearer token")

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadGateway {
		t.Fatalf("expected status 502, got %d", rr.Code)
	}
}

func TestContactsUnauthorized(t *testing.T) {
	h := NewHandler(stubReadiness{ready: true}, &stubSender{}, stubContacts{}, stubMessages{}, stubLogStore{}, Config{AuthToken: "token", GroupJID: "1@g.us"})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/contacts", nil)

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rr.Code)
	}
}

func TestContactsSuccess(t *testing.T) {
	h := NewHandler(
		stubReadiness{ready: true},
		&stubSender{},
		stubContacts{items: []wa.ContactSummary{{JID: "123@s.whatsapp.net", PushName: "Alice"}}},
		stubMessages{},
		stubLogStore{},
		Config{AuthToken: "token", GroupJID: "1@g.us"},
	)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/contacts", nil)
	req.Header.Set("Authorization", "Bearer token")

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
}

func TestMessagesUnauthorized(t *testing.T) {
	h := NewHandler(stubReadiness{ready: true}, &stubSender{}, stubContacts{}, stubMessages{}, stubLogStore{}, Config{AuthToken: "token", GroupJID: "1@g.us"})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/messages", nil)

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rr.Code)
	}
}

func TestMessagesSuccess(t *testing.T) {
	h := NewHandler(
		stubReadiness{ready: true},
		&stubSender{},
		stubContacts{},
		stubMessages{items: []wa.MessageSummary{{ID: "abc", ChatJID: "120@g.us", SenderJID: "628@s.whatsapp.net", Text: "hello", Timestamp: "2026-01-01T00:00:00Z"}}},
		stubLogStore{},
		Config{AuthToken: "token", GroupJID: "1@g.us"},
	)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/messages?limit=1", nil)
	req.Header.Set("Authorization", "Bearer token")

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
}
