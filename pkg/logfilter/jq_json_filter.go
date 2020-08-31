package logfilter

import (
	"encoding/json"

	"github.com/itchyny/gojq"
	"golang.org/x/xerrors"
)

type JQJSONFilter struct {
	Code *gojq.Code
}

func NewJQJSONFilter(queryStr string) (*JQJSONFilter, error) {
	query, err := gojq.Parse(queryStr)
	if err != nil {
		return nil, xerrors.Errorf("failed to parse JQ query: %s: %w", query, err)
	}
	code, err := gojq.Compile(query)
	if err != nil {
		return nil, xerrors.Errorf("failed to compile JQ query: %s: %w", query, err)
	}

	return &JQJSONFilter{
		Code: code,
	}, nil
}

func (f *JQJSONFilter) IsIncluded(b []byte) (bool, error) {
	var input map[string]interface{}

	err := json.Unmarshal(b, &input)
	if err != nil {
		return false, xerrors.Errorf("failed to parse json: %s: %w", string(b), err)
	}

	iter := f.Code.Run(input)

	v, ok := iter.Next()
	if !ok {
		return false, nil
	}
	if err, ok := v.(error); ok {
		return false, err
	}

	return true, nil
}
