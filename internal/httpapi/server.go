package httpapi

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"wa-bot-notif/internal/wa"
)

type Readiness interface {
	Ready() bool
}

type Sender interface {
	SendText(ctx context.Context, jid, text string) error
}

type ContactReader interface {
	ListContacts(ctx context.Context) ([]wa.ContactSummary, error)
}

type MessageReader interface {
	ListMessages(limit int) []wa.MessageSummary
}

type LogStore interface {
	InsertSend(ctx context.Context, message, groupID string) error
	InsertUnauthorized(ctx context.Context, ip, reason string, headers map[string][]string) error
}

type Config struct {
	AuthToken string
	GroupJID  string
}

type statusResponse struct {
	Status string `json:"status"`
}

type sendRequest struct {
	Message string `json:"message"`
	UserID  string `json:"userId,omitempty"`
	GroupID string `json:"groupId,omitempty"`
}

type sendResponse struct {
	Success   bool   `json:"success"`
	SentTo    string `json:"sent_to"`
	Timestamp string `json:"timestamp"`
}

func NewHandler(readiness Readiness, sender Sender, contacts ContactReader, messages MessageReader, logStore LogStore, cfg Config) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, statusResponse{Status: "ok"})
	})

	mux.HandleFunc("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		if readiness != nil && readiness.Ready() {
			writeJSON(w, http.StatusOK, statusResponse{Status: "ready"})
			return
		}
		writeJSON(w, http.StatusServiceUnavailable, statusResponse{Status: "not_ready"})
	})

	mux.HandleFunc("/send", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			writeJSON(w, http.StatusMethodNotAllowed, statusResponse{Status: "method_not_allowed"})
			return
		}

		timestamp := time.Now().UTC().Format(time.RFC3339)

		if !isAuthorized(r.Header.Get("Authorization"), cfg.AuthToken) {
			if err := logStore.InsertUnauthorized(r.Context(), clientIP(r), "Unauthorized", r.Header); err != nil {
				log.Warn().Err(err).Msg("failed to log unauthorized request")
			}
			writeJSON(w, http.StatusUnauthorized, sendResponse{Success: false, SentTo: "unauthorized", Timestamp: timestamp})
			return
		}

		var req sendRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, sendResponse{Success: false, SentTo: "invalid_payload", Timestamp: timestamp})
			return
		}

		message := strings.TrimSpace(req.Message)
		if message == "" {
			writeJSON(w, http.StatusBadRequest, sendResponse{Success: false, SentTo: "invalid_payload", Timestamp: timestamp})
			return
		}

		targetID := resolveSendTarget(req.UserID, req.GroupID, cfg.GroupJID)
		if targetID == "" {
			writeJSON(w, http.StatusBadRequest, sendResponse{Success: false, SentTo: "missing_target", Timestamp: timestamp})
			return
		}

		sendCtx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		err := sender.SendText(sendCtx, targetID, message)
		if err != nil {
			status := http.StatusBadGateway
			if errors.Is(err, wa.ErrNotReady) {
				status = http.StatusServiceUnavailable
			}
			log.Error().Err(err).Str("target", targetID).Msg("send failed")
			writeJSON(w, status, sendResponse{Success: false, SentTo: targetID, Timestamp: timestamp})
			return
		}

		if err := logStore.InsertSend(r.Context(), message, targetID); err != nil {
			log.Warn().Err(err).Msg("failed to log send")
		}
		writeJSON(w, http.StatusOK, sendResponse{Success: true, SentTo: targetID, Timestamp: timestamp})
	})

	mux.HandleFunc("/contacts", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			writeJSON(w, http.StatusMethodNotAllowed, statusResponse{Status: "method_not_allowed"})
			return
		}

		if !isAuthorized(r.Header.Get("Authorization"), cfg.AuthToken) {
			if err := logStore.InsertUnauthorized(r.Context(), clientIP(r), "Unauthorized", r.Header); err != nil {
				log.Warn().Err(err).Msg("failed to log unauthorized request")
			}
			writeJSON(w, http.StatusUnauthorized, sendResponse{Success: false, SentTo: "unauthorized", Timestamp: time.Now().UTC().Format(time.RFC3339)})
			return
		}

		if contacts == nil {
			writeJSON(w, http.StatusServiceUnavailable, statusResponse{Status: "contacts_unavailable"})
			return
		}

		items, err := contacts.ListContacts(r.Context())
		if err != nil {
			writeJSON(w, http.StatusBadGateway, statusResponse{Status: "contacts_read_failed"})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"count":    len(items),
			"contacts": items,
		})
	})

	mux.HandleFunc("/messages", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			writeJSON(w, http.StatusMethodNotAllowed, statusResponse{Status: "method_not_allowed"})
			return
		}

		if !isAuthorized(r.Header.Get("Authorization"), cfg.AuthToken) {
			if err := logStore.InsertUnauthorized(r.Context(), clientIP(r), "Unauthorized", r.Header); err != nil {
				log.Warn().Err(err).Msg("failed to log unauthorized request")
			}
			writeJSON(w, http.StatusUnauthorized, sendResponse{Success: false, SentTo: "unauthorized", Timestamp: time.Now().UTC().Format(time.RFC3339)})
			return
		}

		if messages == nil {
			writeJSON(w, http.StatusServiceUnavailable, statusResponse{Status: "messages_unavailable"})
			return
		}

		limit := 100
		if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
			parsed, err := strconv.Atoi(raw)
			if err != nil || parsed <= 0 {
				writeJSON(w, http.StatusBadRequest, statusResponse{Status: "invalid_limit"})
				return
			}
			limit = parsed
		}

		items := messages.ListMessages(limit)
		writeJSON(w, http.StatusOK, map[string]any{
			"count":    len(items),
			"messages": items,
		})
	})

	return mux
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Warn().Err(err).Msg("failed to encode JSON response")
	}
}

func isAuthorized(headerValue, authToken string) bool {
	expected := "Bearer " + strings.TrimSpace(authToken)
	return subtle.ConstantTimeCompare([]byte(headerValue), []byte(expected)) == 1
}

func clientIP(r *http.Request) string {
	forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For"))
	if forwarded != "" {
		return strings.Split(forwarded, ",")[0]
	}
	return r.RemoteAddr
}

func resolveSendTarget(userID, groupID, fallbackGroup string) string {
	userID = strings.TrimSpace(userID)
	if userID != "" {
		if strings.Contains(userID, "@") {
			return userID
		}

		digits := onlyDigits(userID)
		if digits == "" {
			return ""
		}
		return digits + "@s.whatsapp.net"
	}

	groupID = strings.TrimSpace(groupID)
	if groupID != "" {
		return groupID
	}

	return strings.TrimSpace(fallbackGroup)
}

func onlyDigits(v string) string {
	if v == "" {
		return ""
	}

	buf := make([]rune, 0, len(v))
	for _, r := range v {
		if r >= '0' && r <= '9' {
			buf = append(buf, r)
		}
	}

	return string(buf)
}
