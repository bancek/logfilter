# logfilter

Logfilter is JSON log filtering helper. It can either spawn a command or filter
stdin. It uses Go text/template to filter the JSON log lines. The original
output can either be discarded or written into log-rotated files.

## Running

```sh
export LOGFILTER_EXCLUDETEMPLATE='`{{with .Level}}{{eq . "Debug"}}{{end}}{{with .MessageTemplate}}{{eq . "Ignore this message"}}{{end}}`'
export LOGFILTER_FULLOUTPUTFILENAME="logfilter.log"

go run . sh -c 'echo "{\"Level\": \"Debug\", \"Message\": \"Excluded\"}"; echo "Included"; sleep 2'
```

## Testing

```sh
go test ./...
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
