package main

import (
	"net/url"

	"fmt"
	"github.com/dcu/go-authy"
	"github.com/sfreiberg/gotwilio"
)

// MessageServicer is out interface for sending messages. This can be mocked in the tests.
type MessageServicer interface {
	phoneVerifier
	smsMessager
}

type phoneVerifier interface {
	RequestCode(phoneNumber string) (bool, error)
	VerifyCode(phoneNumber, code string) (bool, error)
}

type smsMessager interface {
	Send(from, to, body string) error
}

type twilioMessageService struct {
	authy  *authy.Authy
	twilio *gotwilio.Twilio
}

func (t *twilioMessageService) Send(from, to, body string) error {
	_, _, err := t.twilio.SendSMS("+1"+from, "+1"+to, body, "", "") // todo: should not ignore those returns
	return err
}

func (t *twilioMessageService) RequestCode(phoneNumber string) (bool, error) {
	fmt.Println("phoneNumber: ", phoneNumber)
	verification, err := t.authy.StartPhoneVerification(1, phoneNumber, "sms", url.Values{})
	return verification.Success, err
}

func (t *twilioMessageService) VerifyCode(phoneNumber, code string) (bool, error) {
	verification, err := t.authy.CheckPhoneVerification(1, phoneNumber, code, url.Values{})
	return verification.Success, err
}
