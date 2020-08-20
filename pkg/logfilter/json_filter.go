package logfilter

import (
	"bytes"
	"encoding/json"
	"text/template"

	"golang.org/x/xerrors"
)

type JSONFilter struct {
	ExcludeTpl *template.Template
	Buf        bytes.Buffer
}

func NewJSONFilter(excludeTemplate string) (*JSONFilter, error) {
	excludeTpl := template.New("exclude")

	_, err := excludeTpl.Parse(excludeTemplate)
	if err != nil {
		return nil, xerrors.Errorf("failed to parse exclude template: %s: %w", excludeTemplate, err)
	}

	return &JSONFilter{
		ExcludeTpl: excludeTpl,
	}, nil
}

func (f *JSONFilter) IsIncluded(b []byte) (bool, error) {
	var v interface{}

	err := json.Unmarshal(b, &v)
	if err != nil {
		return false, xerrors.Errorf("failed to parse json: %s: %w", string(b), err)
	}

	f.Buf.Reset()

	if err := f.ExcludeTpl.Execute(&f.Buf, v); err != nil {
		return false, xerrors.Errorf("failed to execute exclude template: %s: %w", string(b), err)
	}

	exclude := bytes.Contains(f.Buf.Bytes(), []byte{'t', 'r', 'u', 'e'})

	return !exclude, nil
}
