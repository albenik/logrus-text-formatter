package textformatter

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/mgutz/ansi"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh/terminal"
)

type ColorScheme struct {
	InfoLevel  string
	WarnLevel  string
	ErrorLevel string
	FatalLevel string
	PanicLevel string
	DebugLevel string
	Tag        string
	Prefix     string
	Timestamp  string
}

type compiledColorScheme struct {
	InfoLevel  func(string) string
	WarnLevel  func(string) string
	ErrorLevel func(string) string
	FatalLevel func(string) string
	PanicLevel func(string) string
	DebugLevel func(string) string
	Tag        func(string) string
	Prefix     func(string) string
	Timestamp  func(string) string
}

type Instance struct {
	// Use colors if TTY detected
	UseColors bool

	// Disable timestamp logging. useful when output is redirected to logging
	// system that already adds timestamps.
	DisableTimestamp bool

	// Print level names in `lowercase` instead of `UPPERCASE`
	LowercaseLevels bool

	// Enable logging the full timestamp when a TTY is attached instead of just
	// the time passed since beginning of execution.
	FullTimestamp bool

	// Timestamp format to use for display when a full timestamp is printed.
	TimestampFormat string

	colorScheme *compiledColorScheme
}

const defaultTimestampFormat = time.RFC3339Nano

var (
	baseTimestamp time.Time    = time.Now()
	defaultColors *ColorScheme = &ColorScheme{
		InfoLevel:  "green+h",
		WarnLevel:  "yellow+h",
		ErrorLevel: "red+h",
		FatalLevel: "red+h",
		PanicLevel: "red+h",
		DebugLevel: "black+h",
		Tag:        "magenta",
		Prefix:     "cyan",
		Timestamp:  "black+h",
	}
	noColors *compiledColorScheme = &compiledColorScheme{
		InfoLevel:  ansi.ColorFunc(""),
		WarnLevel:  ansi.ColorFunc(""),
		ErrorLevel: ansi.ColorFunc(""),
		FatalLevel: ansi.ColorFunc(""),
		PanicLevel: ansi.ColorFunc(""),
		DebugLevel: ansi.ColorFunc(""),
		Tag:        ansi.ColorFunc(""),
		Prefix:     ansi.ColorFunc(""),
		Timestamp:  ansi.ColorFunc(""),
	}
	defaultCompiledColorScheme *compiledColorScheme = compileColorScheme(defaultColors)
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
		InfoLevel:  getCompiledColor(s.InfoLevel, defaultColors.InfoLevel),
		WarnLevel:  getCompiledColor(s.WarnLevel, defaultColors.WarnLevel),
		ErrorLevel: getCompiledColor(s.ErrorLevel, defaultColors.ErrorLevel),
		FatalLevel: getCompiledColor(s.FatalLevel, defaultColors.FatalLevel),
		PanicLevel: getCompiledColor(s.PanicLevel, defaultColors.PanicLevel),
		DebugLevel: getCompiledColor(s.DebugLevel, defaultColors.DebugLevel),
		Tag:        getCompiledColor(s.Tag, defaultColors.Tag),
		Prefix:     getCompiledColor(s.Prefix, defaultColors.Prefix),
		Timestamp:  getCompiledColor(s.Timestamp, defaultColors.Timestamp),
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
	if entry.Buffer != nil {
		buf = entry.Buffer
	} else {
		buf = &bytes.Buffer{}
	}

	timestampFormat := f.TimestampFormat
	if timestampFormat == "" {
		timestampFormat = defaultTimestampFormat
	}
	var colors *compiledColorScheme
	if f.UseColors {
		if f.colorScheme == nil {
			f.colorScheme = defaultCompiledColorScheme
		}
		colors = f.colorScheme
	} else {
		colors = noColors
	}
	var levelColor func(string) string
	var levelText string
	switch entry.Level {
	case logrus.InfoLevel:
		levelColor = colors.InfoLevel
	case logrus.WarnLevel:
		levelColor = colors.WarnLevel
	case logrus.ErrorLevel:
		levelColor = colors.ErrorLevel
	case logrus.FatalLevel:
		levelColor = colors.FatalLevel
	case logrus.PanicLevel:
		levelColor = colors.PanicLevel
	default:
		levelColor = colors.DebugLevel
	}

	if entry.Level != logrus.WarnLevel {
		levelText = entry.Level.String()
	} else {
		levelText = "warn"
	}

	if !f.LowercaseLevels {
		levelText = strings.ToUpper(levelText)
	}

	if !f.DisableTimestamp {
		var ts string
		if !f.FullTimestamp {
			ts = fmt.Sprintf("[%f]", miniTS())
		} else {
			ts = entry.Time.Format(timestampFormat)
		}
		fmt.Fprint(buf, colors.Timestamp(ts), " ")
	}

	fmt.Fprint(buf, levelColor(fmt.Sprintf("%5s", levelText)), " ")

	if v, ok := entry.Data["__t"]; ok {
		fmt.Fprint(buf, colors.Tag(fmt.Sprintf("%v", v)), " ")
	} else {
		fmt.Fprint(buf, colors.Tag("               -"), " ")
	}

	if v, ok := entry.Data["__p"]; ok {
		fmt.Fprint(buf, colors.Prefix(fmt.Sprintf("%v", v)))
	} else {
		fmt.Fprint(buf, colors.Prefix("__p<missing>"))
	}

	if v, ok := entry.Data["__f"]; ok {
		fmt.Fprint(buf, colors.Prefix(fmt.Sprintf(":%v", v)))
	}
	fmt.Fprint(buf, ": ", entry.Message)

	for k, v := range entry.Data {
		if k != "__p" && k != "__f" && k != "__t" {
			if w, ok := v.(fmt.Stringer); ok {
				fmt.Fprintf(buf, " %s=%s", colors.Prefix(k), w.String())
			} else {
				fmt.Fprintf(buf, " %s=%#v", colors.Prefix(k), v)
			}
		}
	}
	buf.WriteRune('\n')
	return buf.Bytes(), nil
}
