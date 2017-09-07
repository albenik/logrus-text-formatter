package textformatter

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/mgutz/ansi"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh/terminal"
)

type ColorScheme struct {
	InfoLevelStyle  string
	WarnLevelStyle  string
	ErrorLevelStyle string
	FatalLevelStyle string
	PanicLevelStyle string
	DebugLevelStyle string
	PrefixStyle1    string
	PrefixStyle2    string
	TimestampStyle  string
}

type compiledColorScheme struct {
	InfoLevelColor  func(string) string
	WarnLevelColor  func(string) string
	ErrorLevelColor func(string) string
	FatalLevelColor func(string) string
	PanicLevelColor func(string) string
	DebugLevelColor func(string) string
	PrefixColor1    func(string) string
	PrefixColor2    func(string) string
	TimestampColor  func(string) string
}

type Instance struct {
	// Set to true to bypass checking for a TTY before outputting colors.
	ForceColors bool

	// Force disabling colors. For a TTY colors are enabled by default.
	DisableColors bool

	// Force formatted layout, even for non-TTY output.
	ForceFormatting bool

	// Disable timestamp logging. useful when output is redirected to logging
	// system that already adds timestamps.
	DisableTimestamp bool

	// Disable the conversion of the log levels to uppercase
	DisableUppercase bool

	// Enable logging the full timestamp when a TTY is attached instead of just
	// the time passed since beginning of execution.
	FullTimestamp bool

	// Timestamp format to use for display when a full timestamp is printed.
	TimestampFormat string

	// Wrap empty fields in quotes if true.
	QuoteEmptyFields bool

	// Can be set to the override the default quoting character "
	// with something else. For example: ', or `.
	QuoteCharacter string

	// Color scheme to use.
	colorScheme *compiledColorScheme

	// Whether the logger's out is to a terminal.
	isTerminal bool

	sync.Once
}

const defaultTimestampFormat = time.RFC3339Nano

var (
	baseTimestamp      time.Time    = time.Now()
	defaultColorScheme *ColorScheme = &ColorScheme{
		InfoLevelStyle:  "green+h",
		WarnLevelStyle:  "yellow+h",
		ErrorLevelStyle: "red+h",
		FatalLevelStyle: "red+h",
		PanicLevelStyle: "red+h",
		DebugLevelStyle: "black+h",
		PrefixStyle1:    "black+h",
		PrefixStyle2:    "cyan",
		TimestampStyle:  "black+h",
	}
	noColorsColorScheme *compiledColorScheme = &compiledColorScheme{
		InfoLevelColor:  ansi.ColorFunc(""),
		WarnLevelColor:  ansi.ColorFunc(""),
		ErrorLevelColor: ansi.ColorFunc(""),
		FatalLevelColor: ansi.ColorFunc(""),
		PanicLevelColor: ansi.ColorFunc(""),
		DebugLevelColor: ansi.ColorFunc(""),
		PrefixColor1:    ansi.ColorFunc(""),
		PrefixColor2:    ansi.ColorFunc(""),
		TimestampColor:  ansi.ColorFunc(""),
	}
	defaultCompiledColorScheme *compiledColorScheme = compileColorScheme(defaultColorScheme)
)

func miniTS() float64 {
	return time.Since(baseTimestamp).Seconds()
}

func getCompiledColor(main string, fallback string) func(string) string {
	var style string
	if main != "" {
		style = main
	} else {
		style = fallback
	}
	return ansi.ColorFunc(style)
}

func compileColorScheme(s *ColorScheme) *compiledColorScheme {
	return &compiledColorScheme{
		InfoLevelColor:  getCompiledColor(s.InfoLevelStyle, defaultColorScheme.InfoLevelStyle),
		WarnLevelColor:  getCompiledColor(s.WarnLevelStyle, defaultColorScheme.WarnLevelStyle),
		ErrorLevelColor: getCompiledColor(s.ErrorLevelStyle, defaultColorScheme.ErrorLevelStyle),
		FatalLevelColor: getCompiledColor(s.FatalLevelStyle, defaultColorScheme.FatalLevelStyle),
		PanicLevelColor: getCompiledColor(s.PanicLevelStyle, defaultColorScheme.PanicLevelStyle),
		DebugLevelColor: getCompiledColor(s.DebugLevelStyle, defaultColorScheme.DebugLevelStyle),
		PrefixColor1:    getCompiledColor(s.PrefixStyle1, defaultColorScheme.PrefixStyle1),
		PrefixColor2:    getCompiledColor(s.PrefixStyle2, defaultColorScheme.PrefixStyle2),
		TimestampColor:  getCompiledColor(s.TimestampStyle, defaultColorScheme.TimestampStyle),
	}
}

func (f *Instance) init(entry *logrus.Entry) {
	if len(f.QuoteCharacter) == 0 {
		f.QuoteCharacter = "\""
	}
	if entry.Logger != nil {
		f.isTerminal = f.checkIfTerminal(entry.Logger.Out)
	}
}

func (f *Instance) checkIfTerminal(w io.Writer) bool {
	switch v := w.(type) {
	case *os.File:
		return terminal.IsTerminal(int(v.Fd()))
	default:
		return false
	}
}

func (f *Instance) SetColorScheme(colorScheme *ColorScheme) {
	f.colorScheme = compileColorScheme(colorScheme)
}

func (f *Instance) Format(entry *logrus.Entry) ([]byte, error) {
	var buf *bytes.Buffer
	var keys []string = make([]string, 0, len(entry.Data))
	for k := range entry.Data {
		keys = append(keys, k)
	}
	lastKeyIdx := len(keys) - 1

	if entry.Buffer != nil {
		buf = entry.Buffer
	} else {
		buf = &bytes.Buffer{}
	}

	prefixFieldClashes(entry.Data)

	f.Do(func() { f.init(entry) })

	isFormatted := f.ForceFormatting || f.isTerminal

	timestampFormat := f.TimestampFormat
	if timestampFormat == "" {
		timestampFormat = defaultTimestampFormat
	}
	if isFormatted {
		isColored := (f.ForceColors || f.isTerminal) && !f.DisableColors
		var colorScheme *compiledColorScheme
		if isColored {
			if f.colorScheme == nil {
				colorScheme = defaultCompiledColorScheme
			} else {
				colorScheme = f.colorScheme
			}
		} else {
			colorScheme = noColorsColorScheme
		}
		f.printColored(buf, entry, keys, timestampFormat, colorScheme)
	} else {
		if !f.DisableTimestamp {
			f.appendKeyValue(buf, "time", entry.Time.Format(timestampFormat), true)
		}
		f.appendKeyValue(buf, "level", entry.Level.String(), true)
		if entry.Message != "" {
			f.appendKeyValue(buf, "msg", entry.Message, lastKeyIdx >= 0)
		}
		for i, key := range keys {
			f.appendKeyValue(buf, key, entry.Data[key], lastKeyIdx != i)
		}
	}

	buf.WriteRune('\n')
	return buf.Bytes(), nil
}

func (f *Instance) printColored(buf *bytes.Buffer, entry *logrus.Entry, keys []string, timestampFormat string, colorScheme *compiledColorScheme) {
	var levelColor func(string) string
	var levelText string
	switch entry.Level {
	case logrus.InfoLevel:
		levelColor = colorScheme.InfoLevelColor
	case logrus.WarnLevel:
		levelColor = colorScheme.WarnLevelColor
	case logrus.ErrorLevel:
		levelColor = colorScheme.ErrorLevelColor
	case logrus.FatalLevel:
		levelColor = colorScheme.FatalLevelColor
	case logrus.PanicLevel:
		levelColor = colorScheme.PanicLevelColor
	default:
		levelColor = colorScheme.DebugLevelColor
	}

	if entry.Level != logrus.WarnLevel {
		levelText = entry.Level.String()
	} else {
		levelText = "warn"
	}

	if !f.DisableUppercase {
		levelText = strings.ToUpper(levelText)
	}

	if !f.DisableTimestamp {
		var ts string
		if !f.FullTimestamp {
			ts = fmt.Sprintf("[%f]", miniTS())
		} else {
			ts = entry.Time.Format(timestampFormat)
		}
		fmt.Fprint(buf, colorScheme.TimestampColor(ts), " ")
	}

	fmt.Fprint(buf, levelColor(fmt.Sprintf("%5s", levelText)), " ")

	if v, ok := entry.Data["__t"]; ok {
		fmt.Fprint(buf, colorScheme.PrefixColor1(fmt.Sprintf(":%v:", v)), " ")
	}

	if v, ok := entry.Data["__p"]; ok {
		fmt.Fprint(buf, colorScheme.PrefixColor2(fmt.Sprintf("%v", v)))
	} else {
		fmt.Fprint(buf, colorScheme.PrefixColor2("__p<missing>"))
	}

	if v, ok := entry.Data["__f"]; ok {
		fmt.Fprint(buf, colorScheme.PrefixColor2(fmt.Sprintf(".%v", v)))
	}
	fmt.Fprint(buf, ": ", entry.Message)

	for _, k := range keys {
		if k != "__p" && k != "__f" && k != "__t" {
			v := entry.Data[k]
			fmt.Fprintf(buf, " %s=%+v", colorScheme.PrefixColor2(k), v)
		}
	}
}

func (f *Instance) appendKeyValue(b *bytes.Buffer, key string, value interface{}, appendSpace bool) {
	b.WriteString(key)
	b.WriteByte('=')
	f.appendValue(b, value)

	if appendSpace {
		b.WriteByte(' ')
	}
}

func (f *Instance) appendValue(b *bytes.Buffer, value interface{}) {
	switch value := value.(type) {
	case string:
		b.WriteString(value)
	case error:
		errmsg := value.Error()
		b.WriteString(errmsg)
	default:
		fmt.Fprint(b, value)
	}
}

// This is to not silently overwrite `time`, `msg` and `level` fields when
// dumping it. If this code wasn't there doing:
//
//  logrus.WithField("level", 1).Info("hello")
//
// would just silently drop the user provided level. Instead with this code
// it'll be logged as:
//
//  {"level": "info", "fields.level": 1, "msg": "hello", "time": "..."}
func prefixFieldClashes(data logrus.Fields) {
	if t, ok := data["time"]; ok {
		data["fields.time"] = t
	}

	if m, ok := data["msg"]; ok {
		data["fields.msg"] = m
	}

	if l, ok := data["level"]; ok {
		data["fields.level"] = l
	}
}
