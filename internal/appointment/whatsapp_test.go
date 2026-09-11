package appointment

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizePhone(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"digits only", "521234567890", "521234567890"},
		{"with leading plus", "+521234567890", "521234567890"},
		{"with dashes", "+52-123-456-7890", "521234567890"},
		{"with spaces", "+52 123 456 7890", "521234567890"},
		{"with parens", "+52 (123) 456-7890", "521234567890"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizePhone(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestWhatsAppLink(t *testing.T) {
	tests := []struct {
		name    string
		phone   string
		message string
		want    string
	}{
		{
			name:  "without message",
			phone: "+521234567890",
			want:  "https://wa.me/521234567890",
		},
		{
			name:    "with message",
			phone:   "+521234567890",
			message: "Hola, tengo un turno",
			want:    "https://wa.me/521234567890?text=Hola%2C+tengo+un+turno",
		},
		{
			name:  "empty phone",
			phone: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := WhatsAppLink(tt.phone, tt.message)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAppointmentWhatsAppLink(t *testing.T) {
	link := AppointmentWhatsAppLink("+521234567890", "Juan", "2025-01-15", "10:00")
	assert.Contains(t, link, "https://wa.me/521234567890")
	assert.Contains(t, link, "Juan")
	assert.Contains(t, link, "2025-01-15")
	assert.Contains(t, link, "10%3A00")
}

func TestAppointmentWhatsAppLink_NoClientName(t *testing.T) {
	link := AppointmentWhatsAppLink("+521234567890", "", "2025-01-15", "10:00")
	assert.Contains(t, link, "https://wa.me/521234567890")
	assert.NotContains(t, link, "soy")
	assert.Contains(t, link, "2025-01-15")
}
