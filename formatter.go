package selfformatter

import (
	"bytes"
	"fmt"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh/terminal"
	"io"
	"os"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

type fieldKey string

type FieldMap map[fieldKey]string

func (f FieldMap) resolve(key fieldKey) string {
	if k, ok := f[key]; ok {
		return k
	}
	return string(key)
}

type Fields map[string]interface{}

const (
	red                    = 31
	yellow                 = 33
	blue                   = 36
	gray                   = 34
	defaultTimestampFormat = time.RFC3339
	FieldKeyMsg            = "msg"
	FieldKeyLevel          = "level"
	FieldKeyTime           = "time"
	FieldKeyLogrusError    = "logrus_error"
	FieldKeyFunc           = "func"
	FieldKeyFile           = "file"
	Attr1                  = "attr1" // 五个预保留弹性字段，
	Attr2                  = "attr2"
	Attr3                  = "attr3"
	Attr4                  = "attr4"
	Attr5                  = "attr5"
)

var baseTimestamp time.Time

func init() {
	baseTimestamp = time.Now()
}

// this struct is copied from logrus.TextFormatter, but add and delete several fields
type EaseFormatter struct {
	Formatter                 string // definition the log line format, such as "%time% [%level%] [%methodName%] [%kv%] -- %msg%\n"
	KvCom                     string // k, v of fields combine all together by KvCom. such as: type=logrus, default is '='
	FieldMapCom               string // such as type=logrus&api=gin   default is space " "
	ForceColors               bool
	DisableColors             bool
	EnvironmentOverrideColors bool
	FullTimestamp             bool
	DisableTimestamp          bool
	TimestampFormat           string
	DisableSorting            bool
	SortingFunc               func([]string)
	DisableLevelTruncation    bool
	QuoteEmptyFields          bool
	isTerminal                bool
	FieldMap                  FieldMap
	CallerPrettyfier          func(*runtime.Frame) (function string, file string)
	terminalInitOnce          sync.Once
	levelTextMaxLength        int
}

func (f *EaseFormatter) init(entry *logrus.Entry) {
	if entry.Logger != nil {
		f.isTerminal = checkIfTerminal(entry.Logger.Out)
	}
	// Get the max length of the level text
	for _, level := range logrus.AllLevels {
		levelTextLength := utf8.RuneCount([]byte(level.String()))
		if levelTextLength > f.levelTextMaxLength {
			f.levelTextMaxLength = levelTextLength
		}
	}
}

func (f *EaseFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	data := make(Fields)
	for k, v := range entry.Data {
		data[k] = v
	}
	prefixFieldClashes(data, f.FieldMap, entry.HasCaller())
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	var funcVal, fileVal string
	fixedKeys := make([]string, 0, 4+len(data))
	if !f.DisableTimestamp {
		fixedKeys = append(fixedKeys, f.FieldMap.resolve(FieldKeyTime))
	}

	fixedKeys = append(fixedKeys, f.FieldMap.resolve(FieldKeyLevel))
	if entry.Message != "" {
		fixedKeys = append(fixedKeys, f.FieldMap.resolve(FieldKeyMsg))
	}

	if entry.HasCaller() {
		if f.CallerPrettyfier != nil {
			funcVal, fileVal = f.CallerPrettyfier(entry.Caller)
		} else {
			funcVal = entry.Caller.Function
			fileVal = fmt.Sprintf("%s:%d", entry.Caller.File, entry.Caller.Line)
		}

		if funcVal != "" {
			fixedKeys = append(fixedKeys, f.FieldMap.resolve(FieldKeyFunc))
		}
		if fileVal != "" {
			fixedKeys = append(fixedKeys, f.FieldMap.resolve(FieldKeyFile))
		}
	}

	if !f.DisableSorting {
		if f.SortingFunc == nil {
			sort.Strings(keys)
			fixedKeys = append(fixedKeys, keys...)
		} else {
			if !f.isColored() {
				fixedKeys = append(fixedKeys, keys...)
				f.SortingFunc(fixedKeys)
			} else {
				f.SortingFunc(keys)
			}
		}
	} else {
		fixedKeys = append(fixedKeys, keys...)
	}
	if f.Formatter == "" {
		f.Formatter = "%time% %level% -- %msg% %kv%"
	}
	if f.KvCom == "" {
		f.KvCom = "="
	}
	if f.FieldMapCom == "" {
		f.FieldMapCom = "&"
	}

	var b *bytes.Buffer
	if entry.Buffer != nil {
		b = entry.Buffer
	} else {
		b = &bytes.Buffer{}
	}

	f.terminalInitOnce.Do(func() { f.init(entry) })

	timestampFormat := f.TimestampFormat
	if timestampFormat == "" {
		timestampFormat = defaultTimestampFormat
	}
	if f.isColored() {
		f.printColored(b, entry, keys, data, timestampFormat)
	} else {
		output := f.Formatter
		var kvStr string
		for _, key := range fixedKeys {
			switch {
			case key == f.FieldMap.resolve(FieldKeyTime):
				output = strings.Replace(output, "%time%", entry.Time.Format(timestampFormat), 1)
			case key == f.FieldMap.resolve(FieldKeyLevel):
				output = strings.Replace(output, "%level%", strings.ToUpper(entry.Level.String()), 1)
			case key == f.FieldMap.resolve(FieldKeyMsg):
				output = strings.Replace(output, "%msg%", entry.Message, 1)
			case key == f.FieldMap.resolve(FieldKeyFunc) && entry.HasCaller():
				output = strings.Replace(output, "%funVal%", funcVal, 1)
			case key == f.FieldMap.resolve(FieldKeyFile) && entry.HasCaller():
				output = strings.Replace(output, "%fileVal%", fileVal, 1)
			case key == f.FieldMap.resolve(Attr1):
				output = strings.Replace(output, "%attr1%", fmt.Sprintf("%v", data[key]), 1)
			case key == f.FieldMap.resolve(Attr2):
				output = strings.Replace(output, "%attr2%", fmt.Sprintf("%v", data[key]), 1)
			case key == f.FieldMap.resolve(Attr3):
				output = strings.Replace(output, "%attr3%", fmt.Sprintf("%v", data[key]), 1)
			case key == f.FieldMap.resolve(Attr4):
				output = strings.Replace(output, "%attr4%", fmt.Sprintf("%v", data[key]), 1)
			case key == f.FieldMap.resolve(Attr5):
				output = strings.Replace(output, "%attr5%", fmt.Sprintf("%v", data[key]), 1)
			default:
				if kvStr == "" {
					kvStr = fmt.Sprintf("%s%s%s%s", kvStr, key, f.KvCom, data[key])
				} else {
					kvStr = fmt.Sprintf("%s%s%s%s%s", kvStr, f.FieldMapCom, key, f.KvCom, data[key])
				}
			}
		}
		output = strings.Replace(output, "%kv%", kvStr, 1)
		b.WriteString(output)
	}
	b.WriteByte('\n')
	return b.Bytes(), nil
}

func prefixFieldClashes(data Fields, fieldMap FieldMap, reportCaller bool) {
	timeKey := fieldMap.resolve(FieldKeyTime)
	if t, ok := data[timeKey]; ok {
		data["fields."+timeKey] = t
		delete(data, timeKey)
	}
	msgKey := fieldMap.resolve(FieldKeyMsg)
	if m, ok := data[msgKey]; ok {
		data["fields."+msgKey] = m
		delete(data, msgKey)
	}

	levelKey := fieldMap.resolve(FieldKeyLevel)
	if l, ok := data[levelKey]; ok {
		data["fields."+levelKey] = l
		delete(data, levelKey)
	}

	logrusErrKey := fieldMap.resolve(FieldKeyLogrusError)
	if l, ok := data[logrusErrKey]; ok {
		data["fields."+logrusErrKey] = l
		delete(data, logrusErrKey)
	}

	if reportCaller {
		funcKey := fieldMap.resolve(FieldKeyFunc)
		if l, ok := data[funcKey]; ok {
			data["fields."+funcKey] = l
		}
		fileKey := fieldMap.resolve(FieldKeyFile)
		if l, ok := data[fileKey]; ok {
			data["fields."+fileKey] = l
		}
	}
}

func (f *EaseFormatter) isColored() bool {
	isColored := (f.ForceColors || f.isTerminal) && !f.DisableColors
	if f.EnvironmentOverrideColors {
		if force, ok := os.LookupEnv("CLICOLOR_FORCE"); ok && force != "0" {
			isColored = true
		} else if ok && force == "0" {
			isColored = false
		} else if os.Getenv("CLICOLOR") == "0" {
			isColored = false
		}
	}
	return isColored && !f.DisableColors
}

func (f *EaseFormatter) printColored(b *bytes.Buffer, entry *logrus.Entry, keys []string, data Fields, timestampFormat string) {
	var (
		levelColor int
		levelText  = strings.ToUpper(entry.Level.String())
		kvStr      string
		output     = f.Formatter
	)
	switch entry.Level {
	case logrus.DebugLevel, logrus.TraceLevel:
		levelColor = gray
	case logrus.WarnLevel:
		levelColor = yellow
	case logrus.ErrorLevel, logrus.FatalLevel, logrus.PanicLevel:
		levelColor = red
	default:
		levelColor = blue
	}
	if !f.DisableLevelTruncation {
		levelText = levelText[0:4]
	}
	entry.Message = strings.TrimSuffix(entry.Message, "\n")
	caller := ""
	if entry.HasCaller() {
		funcVal := fmt.Sprintf("%s()", entry.Caller.Function)
		fileVal := fmt.Sprintf("%s:%d", entry.Caller.File, entry.Caller.Line)

		if f.CallerPrettyfier != nil {
			funcVal, fileVal = f.CallerPrettyfier(entry.Caller)
		}

		if fileVal == "" {
			caller = funcVal
		} else if funcVal == "" {
			caller = fileVal
		} else {
			caller = fileVal + " " + funcVal
		}
	}
	output = strings.Replace(output, "%funcVal%", caller, 1)
	output = strings.Replace(output, "%msg%", entry.Message, 1)
	if f.DisableTimestamp {
		output = strings.Replace(output, "%time%", "", 1)
		output = strings.Replace(output, "%level%", fmt.Sprintf("\x1b[%dm%s\x1b[0m", levelColor, levelText), 1)
	} else if !f.FullTimestamp {
		output = strings.Replace(output, "%time%", fmt.Sprintf("%d", entry.Time.Sub(baseTimestamp)/time.Second), 1)
		output = strings.Replace(output, "%level%", fmt.Sprintf("\x1b[%dm%s\x1b[0m", levelColor, levelText), 1)
	} else {
		output = strings.Replace(output, "%time%", entry.Time.Format(timestampFormat), 1)
		output = strings.Replace(output, "%level%", fmt.Sprintf("\x1b[%dm%s\x1b[0m", levelColor, levelText), 1)
	}
	for _, k := range keys {
		v := data[k]
		if match, err := regexp.Match("attr([1-5]{1})$", []byte(k)); match && err == nil {
			output = strings.Replace(output, "%"+k+"%", fmt.Sprintf("%v", v), 1)
			continue
		}
		if kvStr == "" {
			kvStr = fmt.Sprintf("\x1b[%dm%s\x1b[0m%s%v", levelColor, k, f.KvCom, v)
		} else {
			kvStr = fmt.Sprintf("%s%s\x1b[%dm%s\x1b[0m%s%s", kvStr, f.FieldMapCom, levelColor, k, f.KvCom, v)
		}
	}
	output = strings.Replace(output, "%kv%", kvStr, 1)
	b.WriteString(output)
}

func checkIfTerminal(w io.Writer) bool {
	switch v := w.(type) {
	case *os.File:
		return terminal.IsTerminal(int(v.Fd()))
	default:
		return false
	}
}
