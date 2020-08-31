package logfilter_test

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kballard/go-shellquote"
	"github.com/kelseyhightower/envconfig"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"golang.org/x/xerrors"

	. "github.com/bancek/logfilter/pkg/logfilter"
)

var _ = Describe("LogFilter", func() {
	testInputLines := []string{
		`{"Timestamp":"2020-08-18T17:16:36.9975268+00:00","Level":"Information","MessageTemplate":"Test message","Properties":{"DurationMs":1}}`,
		"invalid json",
		`{"Timestamp":"2020-08-18T17:16:37.9975268+00:00","Level":"Debug","MessageTemplate":"Lorem ipsum","Properties":{"DurationMs":1}}`,
		`{"Timestamp":"2020-08-18T17:16:38.9975268+00:00","Level":"Information","MessageTemplate":"Dolor sit amet","Properties":{"DurationMs":1}}`,
	}
	testInput := strings.Join(testInputLines, "\n")
	expectedOutput := []string{
		"invalid json",
		`{"Timestamp":"2020-08-18T17:16:38.9975268+00:00","Level":"Information","MessageTemplate":"Dolor sit amet","Properties":{"DurationMs":1}}`,
		"",
	}
	defaultExcludeTpl := `{{with .Level}}{{eq . "Debug"}}{{end}}{{with .MessageTemplate}}{{eq . "Test message"}}{{end}}`

	run := func(config *Config, reader io.Reader, writer io.Writer) error {
		logFilter := NewLogFilter(config, reader, writer, Logger)

		ctx, cancel := context.WithCancel(TestCtx)
		defer cancel()

		err := logFilter.Init(ctx)
		Expect(err).NotTo(HaveOccurred())
		defer logFilter.Close()

		return logFilter.Start()
	}

	It("should parse the config", func() {
		prefix := strings.ToUpper("LOGFILTERTEST" + Rand())
		os.Setenv(prefix+"_CMD", `bash -c "echo \"123\""`)
		os.Setenv(prefix+"_CMDSHUTDOWNTIMEOUT", "1s")
		os.Setenv(prefix+"_EXCLUDETEMPLATE", "tpl")
		os.Setenv(prefix+"_DEBUGLISTENADDR", "localhost:1234")
		os.Setenv(prefix+"_FULLOUTPUTFILENAME", "filename")
		os.Setenv(prefix+"_FULLOUTPUTMAXSIZEMB", "2")
		os.Setenv(prefix+"_FULLOUTPUTMAXAGEDAYS", "3")
		os.Setenv(prefix+"_FULLOUTPUTMAXBACKUPS", "4")
		os.Setenv(prefix+"_FULLOUTPUTCOMPRESS", "true")
		os.Setenv(prefix+"_LOGLEVEL", "warn")

		config := &Config{}

		err := envconfig.Process(prefix, config)
		Expect(err).NotTo(HaveOccurred())

		Expect(config).To(Equal(&Config{
			Cmd:                  []string{"bash", "-c", `echo "123"`},
			CmdShutdownTimeout:   1 * time.Second,
			ExcludeTemplate:      "tpl",
			DebugListenAddr:      "localhost:1234",
			FullOutputFilename:   "filename",
			FullOutputMaxSizeMB:  2,
			FullOutputMaxAgeDays: 3,
			FullOutputMaxBackups: 4,
			FullOutputCompress:   true,
			LogLevel:             "warn",
		}))
	})

	It("should filter the input and wait for ctx to be done", func() {
		config := &Config{}
		config.ExcludeTemplate = defaultExcludeTpl

		inputWait := make(chan struct{})
		defer func() {
			inputWait <- struct{}{}
		}()

		reader := io.MultiReader(bytes.NewReader([]byte(testInput+"\n")), funcReader(func(b []byte) (int, error) {
			<-inputWait
			return 0, io.EOF
		}))
		writer := bytes.NewBuffer(nil)

		logFilter := NewLogFilter(config, reader, writer, Logger)

		ctx, cancel := context.WithTimeout(TestCtx, 200*time.Millisecond)
		defer cancel()

		err := logFilter.Init(ctx)
		Expect(err).NotTo(HaveOccurred())
		defer logFilter.Close()

		err = logFilter.Start()
		Expect(err).NotTo(HaveOccurred())

		Expect(strings.Split(writer.String(), "\n")).To(Equal(expectedOutput))
	})

	It("should filter the input", func() {
		config := &Config{}
		config.ExcludeTemplate = defaultExcludeTpl

		reader := bytes.NewReader([]byte(testInput))
		writer := bytes.NewBuffer(nil)

		err := run(config, reader, writer)
		Expect(err).To(HaveOccurred())

		Expect(strings.Split(writer.String(), "\n")).To(Equal(expectedOutput))
	})

	It("should not filter the input", func() {
		config := &Config{}

		reader := bytes.NewReader([]byte(testInput))
		writer := bytes.NewBuffer(nil)

		err := run(config, reader, writer)
		Expect(err).To(HaveOccurred())

		Expect(writer.String()).To(Equal(testInput + "\n"))
	})

	It("should run the command and filter its stdout", func() {
		scriptLines := []string{}
		for _, line := range testInputLines {
			scriptLines = append(scriptLines, "echo "+shellquote.Join(line))
		}
		config := &Config{}
		config.Cmd = []string{"bash", "-c", strings.Join(scriptLines, "\n")}
		config.ExcludeTemplate = defaultExcludeTpl

		writer := bytes.NewBuffer(nil)

		err := run(config, nil, writer)
		Expect(err).To(HaveOccurred())

		Expect(strings.Split(writer.String(), "\n")).To(Equal(expectedOutput))
	})

	It("should run the command and filter its stderr", func() {
		scriptLines := []string{}
		for _, line := range testInputLines {
			scriptLines = append(scriptLines, "echo "+shellquote.Join(line)+" >&2")
		}
		config := &Config{}
		config.Cmd = []string{"bash", "-c", strings.Join(scriptLines, "\n")}
		config.ExcludeTemplate = defaultExcludeTpl

		writer := bytes.NewBuffer(nil)

		err := run(config, nil, writer)
		Expect(err).To(HaveOccurred())

		Expect(strings.Split(writer.String(), "\n")).To(Equal(expectedOutput))
	})

	It("should run the command and filter its stdout even after ctx is done", func() {
		scriptLines := []string{}
		for _, line := range testInputLines {
			scriptLines = append(scriptLines, "echo "+shellquote.Join(line))
		}
		config := &Config{}
		config.Cmd = []string{"bash", "-c", "function onint {\n" + strings.Join(scriptLines, "\n") + "\n}\n trap onint SIGINT\n while true; do sleep 0.1; done"}
		config.CmdShutdownTimeout = 500 * time.Millisecond
		config.ExcludeTemplate = defaultExcludeTpl

		writer := bytes.NewBuffer(nil)

		logFilter := NewLogFilter(config, nil, writer, Logger)

		ctx, cancel := context.WithTimeout(TestCtx, 200*time.Millisecond)
		defer cancel()

		err := logFilter.Init(ctx)
		Expect(err).NotTo(HaveOccurred())
		defer logFilter.Close()

		err = logFilter.Start()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("signal: killed"))

		Expect(strings.Split(writer.String(), "\n")).To(Equal(expectedOutput))
	})

	It("should fail to run a non-existent command", func() {
		config := &Config{}
		config.Cmd = []string{"nonexistentcmd"}
		config.ExcludeTemplate = defaultExcludeTpl

		writer := bytes.NewBuffer(nil)

		err := run(config, nil, writer)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("executable file not found"))
	})

	It("should fail to parse the exclude template", func() {
		config := &Config{}
		config.ExcludeTemplate = "{{invalid"

		logFilter := NewLogFilter(config, nil, bytes.NewBuffer(nil), Logger)

		err := logFilter.Init(TestCtx)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal(`failed to build json filter: failed to parse exclude template: {{invalid: template: exclude:1: function "invalid" not defined`))
	})

	It("should log the failure of executing the exclude template", func() {
		baseLogger := NewLogger()
		Logger = baseLogger.WithFields(logrus.Fields{})
		testHook := test.NewLocal(baseLogger)

		config := &Config{}
		config.ExcludeTemplate = `{{eq .Level "Debug"}}`

		reader := bytes.NewReader([]byte(`{"lvl": "info"}`))
		writer := bytes.NewBuffer(nil)

		err := run(config, reader, writer)
		Expect(err).To(HaveOccurred())

		Expect(writer.String()).To(Equal(`{"lvl": "info"}` + "\n"))

		entryFound := false
		for _, ent := range testHook.AllEntries() {
			if ent.Message == "LogFilter failed to filter line" {
				entryFound = true
			}
		}
		Expect(entryFound).To(BeTrue())
	})

	It("should write full output to a file", func() {
		tmpDir, err := ioutil.TempDir("", "logfilter-test-")
		Expect(err).NotTo(HaveOccurred())
		defer os.RemoveAll(tmpDir)

		config := &Config{}
		config.ExcludeTemplate = defaultExcludeTpl
		config.FullOutputFilename = filepath.Join(tmpDir, "logfilter.log")

		reader := bytes.NewReader([]byte(testInput))
		writer := bytes.NewBuffer(nil)

		err = run(config, reader, writer)
		Expect(err).To(HaveOccurred())

		Expect(strings.Split(writer.String(), "\n")).To(Equal(expectedOutput))

		out, err := ioutil.ReadFile(config.FullOutputFilename)
		Expect(err).NotTo(HaveOccurred())

		Expect(string(out)).To(Equal(testInput + "\n"))
	})

	It("should fail if writer.Write fails", func() {
		config := &Config{}
		config.ExcludeTemplate = defaultExcludeTpl

		reader := bytes.NewReader([]byte(testInput))
		writer := funcWriter(func(b []byte) (int, error) {
			return 0, xerrors.Errorf("custom write error")
		})

		err := run(config, reader, writer)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("custom write error"))
	})

	It("should fail if full writer fails", func() {
		tmpDir, err := ioutil.TempDir("", "logfilter-test-")
		Expect(err).NotTo(HaveOccurred())
		defer os.RemoveAll(tmpDir)
		Expect(os.Mkdir(filepath.Join(tmpDir, "readonly"), 0400)).To(Succeed())

		config := &Config{}
		config.ExcludeTemplate = defaultExcludeTpl
		config.FullOutputFilename = filepath.Join(tmpDir, "readonly", "logfilter.log")

		reader := bytes.NewReader([]byte(testInput))
		writer := bytes.NewBuffer(nil)

		err = run(config, reader, writer)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("error getting log file info"))
	})

	It("should rotate full output file", func() {
		tmpDir, err := ioutil.TempDir("", "logfilter-test-")
		Expect(err).NotTo(HaveOccurred())
		defer os.RemoveAll(tmpDir)

		config := &Config{}
		config.ExcludeTemplate = defaultExcludeTpl
		config.FullOutputFilename = filepath.Join(tmpDir, "logfilter.log")
		config.FullOutputMaxSizeMB = 1

		input := []byte{}
		for len(input) < 1152*1024 {
			input = append(input, []byte(`{"Level": "Debug"}`+"\n")...)
		}
		reader := bytes.NewReader([]byte(input))
		writer := bytes.NewBuffer(nil)

		err = run(config, reader, writer)
		Expect(err).To(HaveOccurred())

		Expect(writer.Bytes()).To(BeEmpty())

		Eventually(func() int {
			items, _ := ioutil.ReadDir(tmpDir)
			rotated := 0
			for _, item := range items {
				if strings.HasPrefix(item.Name(), "logfilter-") && strings.HasSuffix(item.Name(), ".log") {
					rotated++
				}
			}
			return rotated
		}).Should(BeNumerically(">", 0))
	})
})

type funcReader func([]byte) (int, error)

func (r funcReader) Read(b []byte) (int, error) {
	return r(b)
}

type funcWriter func([]byte) (int, error)

func (r funcWriter) Write(b []byte) (int, error) {
	return r(b)
}
