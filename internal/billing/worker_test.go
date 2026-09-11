package billing

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

type mockNotifSender struct {
	sent []NotificationMessage
}

func (m *mockNotifSender) SendToProvider(_ context.Context, _ uuid.UUID, msg NotificationMessage) error {
	m.sent = append(m.sent, msg)
	return nil
}

func TestNewWorker(t *testing.T) {
	w := NewWorker(nil, nil, nil, nil)
	assert.NotNil(t, w)
	assert.Nil(t, w.pool)
	assert.Nil(t, w.service)
	assert.Nil(t, w.repo)
	assert.Nil(t, w.notifService)
}

func TestNewWorker_WithNotifService(t *testing.T) {
	mock := &mockNotifSender{}
	w := NewWorker(nil, nil, nil, mock)
	assert.NotNil(t, w)
	assert.Equal(t, mock, w.notifService)
}

func TestRunInvoiceNotifications_NilNotifService(t *testing.T) {
	w := NewWorker(nil, nil, nil, nil)
	err := w.RunInvoiceNotifications(context.Background())
	assert.NoError(t, err)
}

func TestNotificationMessage_Fields(t *testing.T) {
	msg := NotificationMessage{
		Title: "Test Title",
		Body:  "Test Body",
		Data: map[string]string{
			"type": "test",
		},
	}

	assert.Equal(t, "Test Title", msg.Title)
	assert.Equal(t, "Test Body", msg.Body)
	assert.Equal(t, "test", msg.Data["type"])
}

func TestMockNotifSender_RecordsMessages(t *testing.T) {
	mock := &mockNotifSender{}
	msg := NotificationMessage{
		Title: "Invoice",
		Body:  "You owe $5.00",
		Data:  map[string]string{"type": "billing_invoice"},
	}

	err := mock.SendToProvider(context.Background(), uuid.New(), msg)
	assert.NoError(t, err)
	assert.Len(t, mock.sent, 1)
	assert.Equal(t, "Invoice", mock.sent[0].Title)
}
