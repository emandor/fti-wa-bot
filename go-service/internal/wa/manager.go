package wa

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

var (
	ErrNotReady = errors.New("whatsapp client is not ready")
)

type Manager struct {
	mu sync.RWMutex
	mm sync.RWMutex

	client    *whatsmeow.Client
	container *sqlstore.Container
	state     *State
	messages  []MessageSummary

	reconnecting atomic.Bool
}

type ContactSummary struct {
	JID           string `json:"jid"`
	FirstName     string `json:"first_name,omitempty"`
	FullName      string `json:"full_name,omitempty"`
	PushName      string `json:"push_name,omitempty"`
	BusinessName  string `json:"business_name,omitempty"`
	RedactedPhone string `json:"redacted_phone,omitempty"`
}

type MessageSummary struct {
	ID        string `json:"id"`
	ChatJID   string `json:"chat_jid"`
	SenderJID string `json:"sender_jid"`
	PushName  string `json:"push_name,omitempty"`
	Text      string `json:"text,omitempty"`
	FromMe    bool   `json:"from_me"`
	Timestamp string `json:"timestamp"`
}

const maxRecentMessages = 500

func NewManager(ctx context.Context, sqliteDSN string, state *State) (*Manager, error) {
	if state == nil {
		state = NewState(false)
	}

	dbLog := waLog.Stdout("WA-DB", "INFO", true)
	container, err := sqlstore.New(ctx, "sqlite3", sqliteDSN, dbLog)
	if err != nil {
		return nil, fmt.Errorf("create sqlstore: %w", err)
	}

	deviceStore, err := container.GetFirstDevice(ctx)
	if err != nil {
		return nil, fmt.Errorf("get first device: %w", err)
	}

	clientLog := waLog.Stdout("WA", "INFO", true)
	client := whatsmeow.NewClient(deviceStore, clientLog)

	m := &Manager{
		client:    client,
		container: container,
		state:     state,
	}

	client.AddEventHandler(m.onEvent)

	return m, nil
}

func (m *Manager) Start(ctx context.Context) error {
	if m == nil {
		return errors.New("nil manager")
	}

	client := m.Client()
	if client == nil {
		return errors.New("nil whatsapp client")
	}

	if client.Store.ID == nil {
		qrChan, err := client.GetQRChannel(ctx)
		if err != nil {
			return fmt.Errorf("get qr channel: %w", err)
		}

		go func() {
			for evt := range qrChan {
				if evt.Event == "code" {
					log.Print("[WA] Scan QR code below to pair:")
					printTerminalQR(evt.Code)
				} else {
					log.Printf("[WA] login event: %s", evt.Event)
				}
			}
		}()
	}

	if err := m.connectWithBackoff(ctx); err != nil {
		return err
	}

	return nil
}

func (m *Manager) Shutdown() {
	if m == nil {
		return
	}
	client := m.Client()
	if client != nil {
		client.Disconnect()
	}
	m.state.SetReady(false)
}

func (m *Manager) Ready() bool {
	if m == nil {
		return false
	}
	return m.state.Ready()
}

func (m *Manager) SendText(ctx context.Context, jid, text string) error {
	if !m.Ready() {
		return ErrNotReady
	}

	client := m.Client()
	if client == nil {
		return ErrNotReady
	}

	parsedJID, err := types.ParseJID(jid)
	if err != nil {
		return fmt.Errorf("parse jid: %w", err)
	}

	_, err = client.SendMessage(ctx, parsedJID, &waProto.Message{Conversation: proto.String(text)})
	if err != nil {
		m.state.SetReady(false)
		return fmt.Errorf("send message: %w", err)
	}

	return nil
}

func (m *Manager) ListContacts(ctx context.Context) ([]ContactSummary, error) {
	client := m.Client()
	if client == nil || client.Store == nil || client.Store.Contacts == nil {
		return nil, errors.New("contact store not available")
	}

	contacts, err := client.Store.Contacts.GetAllContacts(ctx)
	if err != nil {
		return nil, fmt.Errorf("get all contacts: %w", err)
	}

	out := make([]ContactSummary, 0, len(contacts))
	for jid, info := range contacts {
		out = append(out, ContactSummary{
			JID:           jid.String(),
			FirstName:     info.FirstName,
			FullName:      info.FullName,
			PushName:      info.PushName,
			BusinessName:  info.BusinessName,
			RedactedPhone: info.RedactedPhone,
		})
	}

	return out, nil
}

func (m *Manager) ListMessages(limit int) []MessageSummary {
	if limit <= 0 || limit > maxRecentMessages {
		limit = 100
	}

	m.mm.RLock()
	defer m.mm.RUnlock()

	if len(m.messages) == 0 {
		return nil
	}

	n := limit
	if n > len(m.messages) {
		n = len(m.messages)
	}

	out := make([]MessageSummary, n)
	copy(out, m.messages[:n])
	return out
}

func (m *Manager) Client() *whatsmeow.Client {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.client
}

func (m *Manager) onEvent(evt interface{}) {
	switch v := evt.(type) {
	case *events.Connected:
		m.state.SetReady(true)
		m.reconnecting.Store(false)
		log.Print("[WA] connected")
	case *events.Disconnected:
		m.state.SetReady(false)
		m.triggerReconnect()
	case *events.LoggedOut:
		m.state.SetReady(false)
		log.Print("[WA] logged out; re-pair required")
	case *events.Message:
		m.addMessage(v)
	}
}

func (m *Manager) addMessage(evt *events.Message) {
	if evt == nil {
		return
	}

	summary := MessageSummary{
		ID:        evt.Info.ID,
		ChatJID:   evt.Info.Chat.String(),
		SenderJID: evt.Info.Sender.String(),
		PushName:  evt.Info.PushName,
		Text:      extractText(evt.Message),
		FromMe:    evt.Info.IsFromMe,
		Timestamp: evt.Info.Timestamp.UTC().Format(time.RFC3339),
	}

	m.mm.Lock()
	defer m.mm.Unlock()

	m.messages = append([]MessageSummary{summary}, m.messages...)
	if len(m.messages) > maxRecentMessages {
		m.messages = m.messages[:maxRecentMessages]
	}
}

func extractText(msg *waProto.Message) string {
	if msg == nil {
		return ""
	}
	if text := msg.GetConversation(); text != "" {
		return text
	}
	if ext := msg.GetExtendedTextMessage(); ext != nil {
		if text := ext.GetText(); text != "" {
			return text
		}
	}
	if image := msg.GetImageMessage(); image != nil {
		if caption := image.GetCaption(); caption != "" {
			return caption
		}
	}
	if video := msg.GetVideoMessage(); video != nil {
		if caption := video.GetCaption(); caption != "" {
			return caption
		}
	}
	return ""
}

func (m *Manager) triggerReconnect() {
	if !m.reconnecting.CompareAndSwap(false, true) {
		return
	}

	go func() {
		defer m.reconnecting.Store(false)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		if err := m.connectWithBackoff(ctx); err != nil {
			log.Printf("[WA] reconnect failed: %v", err)
		}
	}()
}

func (m *Manager) connectWithBackoff(ctx context.Context) error {
	client := m.Client()
	if client == nil {
		return errors.New("nil whatsapp client")
	}

	base := 1 * time.Second
	max := 60 * time.Second

	for attempt := 0; ; attempt++ {
		if err := client.Connect(); err == nil {
			return nil
		} else {
			wait := backoffDuration(base, max, attempt)
			log.Printf("[WA] connect failed (attempt=%d): %v; retrying in %s", attempt+1, err, wait)

			select {
			case <-ctx.Done():
				return fmt.Errorf("connect canceled: %w", ctx.Err())
			case <-time.After(wait):
			}
		}
	}
}

func backoffDuration(base, max time.Duration, attempt int) time.Duration {
	if attempt < 0 {
		attempt = 0
	}

	shift := attempt
	if shift > 6 {
		shift = 6
	}

	wait := base * (1 << shift)
	if wait > max {
		wait = max
	}

	jitterRange := int64(wait / 5)
	if jitterRange <= 0 {
		return wait
	}

	jitter := time.Duration(rand.Int63n(jitterRange))
	return wait - jitter
}
