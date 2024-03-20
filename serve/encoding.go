package serve

import (
	"encoding/json"
	"errors"
	"io"
)

var errNoPost = errors.New("no post provided")
var errBadJson = errors.New("bad JSON")

type incomingReply struct {
	Subject string
	Content string
}

func getIncomingReply(body io.ReadCloser) (*incomingReply, error) {
	if body == nil {
		return nil, errNoPost
	}
	ir := &incomingReply{}
	err := json.NewDecoder(body).Decode(ir)
	if err != nil {
		return nil, errBadJson
	}
	return ir, nil
}

func (ir *incomingReply) Sanitize(uup *incomingReply, isThread bool) error {
	subject, err := checkSubject(uup.Subject, isThread)
	if err != nil {
		return err
	}

	content, err := checkContent(uup.Content)
	if err != nil {
		return err
	}

	ir.Subject = subject
	ir.Content = content
	return nil
}
