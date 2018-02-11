package smtp

import (
	"fmt"
	gosmtp "net/smtp"

	"github.com/getfider/fider/app/pkg/email"
	"github.com/getfider/fider/app/pkg/log"
)

//Sender is used to send e-mails
type Sender struct {
	logger   log.Logger
	host     string
	port     string
	username string
	password string
}

//NewSender creates a new mailgun e-mail sender
func NewSender(logger log.Logger, host, port, username, password string) *Sender {
	return &Sender{logger, host, port, username, password}
}

//Send an e-mail
func (s *Sender) Send(templateName, from string, to email.Recipient) error {
	if !email.CanSendTo(to.Address) {
		s.logger.Warnf("Skipping e-mail to %s due to whitelist.", to.Address)
		return nil
	}

	s.logger.Debugf("Sending e-mail to %s with template %s and params %s.", to.Address, templateName, to.Params)

	message := email.RenderMessage(templateName, to.Params)
	headers := make(map[string]string)
	headers["From"] = fmt.Sprintf("%s <%s>", from, email.NoReply)
	headers["To"] = to.Address
	headers["Subject"] = message.Subject
	headers["MIME-version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=\"UTF-8\""

	body := ""
	for k, v := range headers {
		body += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	body += "\r\n" + message.Body

	servername := fmt.Sprintf("%s:%s", s.host, s.port)
	auth := gosmtp.PlainAuth("", s.username, s.password, s.host)
	err := gosmtp.SendMail(servername, auth, email.NoReply, []string{to.Address}, []byte(body))
	if err != nil {
		s.logger.Errorf("Failed to send e-mail")
		return err
	}
	s.logger.Debugf("E-mail sent.")
	return nil
}

// BatchSend an e-mail to multiple recipients
func (s *Sender) BatchSend(templateName, from string, to []email.Recipient) error {
	for _, r := range to {
		if err := s.Send(templateName, from, r); err != nil {
			return err
		}
	}
	return nil
}