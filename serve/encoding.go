package serve

import (
	"encoding/json"
	"errors"
	"io"
	"spiritchat/validation"
)

var errNoData = errors.New("no data provided")
var errBadJson = errors.New("bad JSON")

type incomingReply struct {
	Subject string `json:"subject"`
	Content string `json:"content"`
}

func getIncomingReply(body io.ReadCloser) (*incomingReply, error) {
	if body == nil {
		return nil, errNoData
	}
	ir := &incomingReply{}
	err := json.NewDecoder(body).Decode(ir)
	if err != nil {
		return nil, errBadJson
	}
	return ir, nil
}

func (ir *incomingReply) Sanitize(isThread bool) error {
	subject, err := validation.ValidateReplySubject(ir.Subject, isThread)
	if err != nil {
		return err
	}

	content, err := validation.ValidateReplyContent(ir.Content)
	if err != nil {
		return err
	}

	ir.Subject = subject
	ir.Content = content
	return nil
}

type incomingSignup struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
}

func (is *incomingSignup) Sanitize() error {
	email, err := validation.ValidateEmail(is.Email)
	if err != nil {
		return err
	}
	password, err := validation.ValidatePassword(is.Password)
	if err != nil {
		return err
	}
	username, err := validation.ValidateUsername(is.Username)
	if err != nil {
		return err
	}
	is.Email = email
	is.Password = password
	is.Username = username
	return nil
}

func getIncomingSignup(body io.ReadCloser) (*incomingSignup, error) {
	if body == nil {
		return nil, errNoData
	}

	is := &incomingSignup{}
	err := json.NewDecoder(body).Decode(is)
	if err != nil {
		return nil, errBadJson
	}
	return is, nil
}
