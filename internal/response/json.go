package response

import (
	"bytes"
	"encoding/json"
	"strconv"
)

type JSONResponse struct {
	Response
}

func NewJSONResponse(data any) (Response, error) {
	body, err := json.Marshal(data)

	if err != nil {
		return nil, err
	}

	br := NewBaseResponse().
		WithHeader("content-type", "application/json").
		WithHeader("content-length", strconv.Itoa(len(body))).
		WithBody(bytes.NewReader(body))

	return &JSONResponse{
		Response: br,
	}, err
}
