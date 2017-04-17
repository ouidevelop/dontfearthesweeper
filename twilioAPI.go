package main

import (
	"net/url"

	"fmt"
	"github.com/dcu/go-authy"
	"github.com/sfreiberg/gotwilio"
)

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

type TwilioMessageService struct {
	authy  *authy.Authy
	twilio *gotwilio.Twilio
}

func (t *TwilioMessageService) Send(from, to, body string) error {
	_, _, err := t.twilio.SendSMS("+1"+from, "+1"+to, body, "", "") // todo: should not ignore those returns
	return err
}

func (t *TwilioMessageService) RequestCode(phoneNumber string) (bool, error) {
	verification, err := t.authy.StartPhoneVerification(1, phoneNumber, "sms", url.Values{})
	return verification.Success, err
}

func (t *TwilioMessageService) VerifyCode(phoneNumber, code string) (bool, error) {
	verification, err := t.authy.CheckPhoneVerification(1, phoneNumber, code, url.Values{})
	fmt.Printf("%+v", verification)
	return verification.Success, err
}
