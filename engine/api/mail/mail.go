package mail

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net/mail"
	"net/smtp"

	"github.com/matcornic/hermes"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

var smtpUser, smtpPassword, smtpFrom, smtpHost, smtpPort string
var smtpTLS, smtpEnable bool
var h hermes.Hermes

// Init initializes configuration
func Init(user, password, from, host, port string, tls, disable bool) {
	smtpUser = user
	smtpPassword = password
	smtpFrom = from
	smtpHost = host
	smtpPort = port
	smtpTLS = tls
	smtpEnable = !disable

	h = hermes.Hermes{
		Product: hermes.Product{
			Name: "CDS",
			Link: "https://github.com/ovh/cds",
			Logo: "https://github.com/ovh/cds/blob/master/cds.png?raw=true",
		},
	}
}

// Status verification of smtp configuration, returns OK or KO
func Status() sdk.MonitoringStatusLine {
	if _, err := smtpClient(); err != nil {
		return sdk.MonitoringStatusLine{Component: "SMTP", Value: "KO: " + err.Error(), Status: sdk.MonitoringStatusAlert}
	}
	return sdk.MonitoringStatusLine{Component: "SMTP", Value: "Connect OK", Status: sdk.MonitoringStatusOK}
}

func smtpClient() (*smtp.Client, error) {
	if smtpHost == "" || smtpPort == "" || !smtpEnable {
		return nil, errors.New("No SMTP configuration")
	}

	// Connect to the SMTP Server
	servername := fmt.Sprintf("%s:%s", smtpHost, smtpPort)

	// TLS config
	tlsconfig := &tls.Config{
		InsecureSkipVerify: false,
		ServerName:         smtpHost,
	}

	var c *smtp.Client
	var err error
	if smtpTLS {
		// Here is the key, you need to call tls.Dial instead of smtp.Dial
		// for smtp servers running on 465 that require an ssl connection
		// from the very beginning (no starttls)
		conn, errc := tls.Dial("tcp", servername, tlsconfig)
		if errc != nil {
			log.Warning("Error with c.Dial:%s\n", errc.Error())
			return nil, errc
		}

		c, err = smtp.NewClient(conn, smtpHost)
		if err != nil {
			log.Warning("Error with c.NewClient:%s\n", err.Error())
			return nil, err
		}
	} else {
		c, err = smtp.Dial(servername)
		if err != nil {
			log.Warning("Error with c.NewClient:%s\n", err.Error())
			return nil, err
		}
	}

	// Auth
	if smtpUser != "" && smtpPassword != "" {
		auth := smtp.PlainAuth("", smtpUser, smtpPassword, smtpHost)
		if err = c.Auth(auth); err != nil {
			log.Warning("Error with c.Auth:%s\n", err.Error())
			c.Close()
			return nil, err
		}
	}
	return c, nil
}

// SendMailVerifyToken Send mail to verify user account
func SendMailVerifyToken(userMail, username, token, callback string) error {
	callbackURL := getCallbackURL(username, token, callback)
	email := hermes.Email{
		Body: hermes.Body{
			Name: "Welcome to CDS",
			Intros: []string{
				"You recently signed up for CDS.",
				"To verify your email address, follow this link : " + callbackURL,
			},
			Outros: []string{
				"Regards",
			},
			Signature: "CDS Team",
		},
	}
	return SendEmail("Welcome to CDS", email, userMail)
}

func getCallbackURL(username, token, callback string) string {
	return fmt.Sprintf(callback, username, token)
}

//SendEmail is the core function to send an email
func SendEmail(subject string, email hermes.Email, userMail string) error {
	from := mail.Address{
		Name:    "",
		Address: smtpFrom,
	}
	to := mail.Address{
		Name:    "",
		Address: userMail,
	}

	// Setup headers
	headers := make(map[string]string)
	headers["From"] = smtpFrom
	headers["To"] = to.String()
	headers["Subject"] = subject

	// Setup message
	message := ""
	for k, v := range headers {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}

	if smtpEnable {
		emailBody, errE := h.GenerateHTML(email)
		if errE != nil {
			return sdk.WrapError(errE, "SendEmail> Unable to generate html email")
		}
		message += "\r\n" + emailBody
	} else {
		emailBody, errE := h.GeneratePlainText(email)
		if errE != nil {
			return sdk.WrapError(errE, "SendEmail> Unable to generate plain text email")
		}
		message += "\r\n" + emailBody
	}

	if !smtpEnable {
		fmt.Println("##### NO SMTP DISPLAY MAIL IN CONSOLE ######")
		fmt.Printf("Subject:%s\n", subject)
		fmt.Printf("Text:%s\n", message)
		fmt.Println("##### END MAIL ######")
		return nil
	}

	c, err := smtpClient()
	if err != nil {
		return sdk.WrapError(err, "Cannot get smtp client")
	}
	defer c.Close()

	// To && From
	if err = c.Mail(from.Address); err != nil {
		return sdk.WrapError(err, "Error with c.Mail")
	}

	if err = c.Rcpt(to.Address); err != nil {
		return sdk.WrapError(err, "Error with c.Rcpt")
	}

	// Data
	w, err := c.Data()
	if err != nil {
		return sdk.WrapError(err, "Error with c.Data")
	}

	_, err = w.Write([]byte(message))
	if err != nil {
		return sdk.WrapError(err, "Error with c.Write")
	}

	err = w.Close()
	if err != nil {
		return sdk.WrapError(err, "Error with c.Close")
	}

	c.Quit()

	return nil
}
