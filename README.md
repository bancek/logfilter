# logfilter

Logfilter is JSON log filtering helper. It can either spawn a command or filter
stdin. It uses Go text/template to filter the JSON log lines. The original
output can either be discarded or written into log-rotated files.

## Running

```sh
export LOGFILTER_EXCLUDETEMPLATE='{{with .Level}}{{eq . "Debug"}}{{end}}{{with .MessageTemplate}}{{eq . "Ignore this message"}}{{end}}'
# or
export LOGFILTER_FILTERQUERY='select(.Level != "Debug") | select(.MessageTemplate != "Test message")'

export LOGFILTER_CMDSHUTDOWNTIMEOUT="10s"
export LOGFILTER_FULLOUTPUTFILENAME="logfilter.log"
export LOGFILTER_FULLOUTPUTMAXSIZEMB="100"
export LOGFILTER_FULLOUTPUTMAXBACKUPS="3"
export LOGFILTER_LOGLEVEL="warn"

go run . sh -c 'echo "{\"Level\": \"Debug\", \"Message\": \"Excluded\"}"; echo "Included"; sleep 2'
```

See [config.go](./pkg/logfilter/config.go) for full configuration.

## Testing

```sh
go test ./...

# Run tests in Docker
docker run --rm -v "$(pwd):/app" -w /app -e CGO_ENABLED=0 golang:1.14.7-alpine sh -c 'go get ./... && go test ./...'
```

### Coverage

```sh
go test --coverprofile logfilter.coverprofile ./pkg/logfilter && go tool cover -html=logfilter.coverprofile -o logfilter.coverprofile.html
```

## Debug

Get a list of goroutines:

```sh
curl localhost:4083/debug/pprof/goroutine?debug=2
```

## Benchmarks

Filtering a single JSON line:

```json
{"Timestamp":"2020-08-18T17:16:36.9975268+00:00","Level":"Information","MessageTemplate":"Test message","Properties":{"DurationMs":1}}
```

JQ filter query:

```
select(.Level != "Debug") | select(.MessageTemplate != "Test message")
```

Exclude template:

```
{{with .Level}}{{eq . "Debug"}}{{end}}{{with .MessageTemplate}}{{eq . "Test message"}}{{end}}
```

Results:

```
BenchmarkJQJSONFilter-16                  178380              5723 ns/op            3032 B/op         61 allocs/op
BenchmarkTemplateJSONFilter-16            241876              4463 ns/op            1937 B/op         43 allocs/op
```