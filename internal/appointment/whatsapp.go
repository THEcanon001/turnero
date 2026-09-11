package appointment

import (
	"fmt"
	"net/url"
	"strings"
)

// WhatsAppLink builds a wa.me deep link with an optional pre-filled message.
func WhatsAppLink(phone, message string) string {
	phone = sanitizePhone(phone)
	if phone == "" {
		return ""
	}

	link := "https://wa.me/" + phone
	if message != "" {
		link += "?text=" + url.QueryEscape(message)
	}
	return link
}

// AppointmentWhatsAppLink builds a WhatsApp link with a pre-filled message
// containing appointment context.
func AppointmentWhatsAppLink(providerPhone, clientName, date, startTime string) string {
	msg := fmt.Sprintf("Hola, tengo un turno el %s a las %s", date, startTime)
	if clientName != "" {
		msg = fmt.Sprintf("Hola, soy %s. Tengo un turno el %s a las %s", clientName, date, startTime)
	}
	return WhatsAppLink(providerPhone, msg)
}

// sanitizePhone removes non-digit chars except leading +.
func sanitizePhone(phone string) string {
	if phone == "" {
		return ""
	}

	var b strings.Builder
	for i, r := range phone {
		if r == '+' && i == 0 {
			continue // skip leading +, wa.me uses raw digits
		}
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
