package notification

import (
	"github.com/matcornic/hermes"

	"github.com/ovh/cds/engine/api/mail"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// SendMailNotif Send user notification by mail
func SendMailNotif(notif sdk.EventNotif) {
	log.Info("notification.SendMailNotif> Send notif '%s'", notif.Subject)
	errors := []string{}

	email := hermes.Email{
		Body: hermes.Body{
			Intros: []string{
				notif.Body,
			},
		},
	}

	for _, recipient := range notif.Recipients {
		if err := mail.SendEmail(notif.Subject, email, recipient); err != nil {
			errors = append(errors, err.Error())
		}
	}
}
