package infobip

import (
	"context"
	"sync"
	"time"

	"github.com/infobip/infobip-api-go-client/v2"
	ib "github.com/infobip/infobip-api-go-client/v2"
	"github.com/sirupsen/logrus"
)

var Debug = false

type Messenger struct {
	client *ib.APIClient

	token       string
	messengerID string
	recipient   string

	mu       sync.RWMutex
	sentAt   time.Time
	cooldown time.Duration
}

func NewMessenger(
	token, host string,
	recipient string,
	messengerID string,
	cooldown time.Duration,
) *Messenger {

	conf := ib.NewConfiguration()
	conf.Host = host

	if Debug {
		conf.Debug = true
	}

	return &Messenger{
		token:       token,
		messengerID: messengerID,
		recipient:   recipient,
		cooldown:    cooldown,
		client:      ib.NewAPIClient(conf),
	}
}

func (m *Messenger) Send(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.sentAt.Add(m.cooldown).After(time.Now()) {
		if Debug {
			logrus.Info("infobip message sending cooldown")
		}

		return nil
	}

	request := infobip.NewSmsAdvancedTextualRequest()

	text := "The proxy is under a heavy load."
	request.SetMessages([]ib.SmsTextualMessage{
		{
			From: &m.messengerID,
			Destinations: &[]infobip.SmsDestination{
				*infobip.NewSmsDestination(m.recipient),
			},
			Text: &text,
		},
	})

	_, _, err := m.client.
		SendSmsApi.
		SendSmsMessage(context.WithValue(
			ctx,
			infobip.ContextAPIKey,
			m.token,
		)).
		SmsAdvancedTextualRequest(*request).
		Execute()

	m.sentAt = time.Now()

	return err
}
