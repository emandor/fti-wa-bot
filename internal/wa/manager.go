package wa

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog/log"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

var ErrNotReady = errors.New("whatsapp client is not ready")

type Manager struct {
	mu sync.RWMutex

	client    *whatsmeow.Client
	container *sqlstore.Container
	state     *State
	messages  *MessageBuffer

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
		messages:  NewMessageBuffer(500),
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
					log.Info().Msg("[WA] Scan QR code below to pair:")
					printTerminalQR(evt.Code)
				} else {
					log.Info().Str("event", evt.Event).Msg("[WA] login event")
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
	return m.messages.List(limit)
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
		log.Info().Msg("[WA] connected")
	case *events.Disconnected:
		m.state.SetReady(false)
		m.triggerReconnect()
	case *events.LoggedOut:
		m.state.SetReady(false)
		log.Warn().Msg("[WA] logged out; re-pair required")
	case *events.Message:
		m.messages.Add(messageFromEvent(v))
	}
}

func messageFromEvent(evt *events.Message) MessageSummary {
	return MessageSummary{
		ID:        evt.Info.ID,
		ChatJID:   evt.Info.Chat.String(),
		SenderJID: evt.Info.Sender.String(),
		PushName:  evt.Info.PushName,
		Text:      extractText(evt.Message),
		FromMe:    evt.Info.IsFromMe,
		Timestamp: evt.Info.Timestamp.UTC().Format(time.RFC3339),
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
			log.Error().Err(err).Msg("[WA] reconnect failed")
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
			log.Warn().Err(err).Int("attempt", attempt+1).Dur("retry_in", wait).Msg("[WA] connect failed")

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
