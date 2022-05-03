package infobip

import (
	"context"

	"github.com/infobip/infobip-api-go-client/v2"
	ib "github.com/infobip/infobip-api-go-client/v2"
)

var Debug = false

type Messenger struct {
	token       string
	messengerID string
	recipient   string

	client *ib.APIClient
}

func NewMessenger(
	token, host string,
	recipient string,
	messengerID string,
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
		client:      ib.NewAPIClient(conf),
	}
}

func (m *Messenger) Send(ctx context.Context) error {
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

	return err
}
