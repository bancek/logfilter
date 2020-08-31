package logfilter

type JSONFilter interface {
	IsIncluded(b []byte) (bool, error)
}

type StaticJSONFilter bool

func (f StaticJSONFilter) IsIncluded(b []byte) (bool, error) {
	return bool(f), nil
}
