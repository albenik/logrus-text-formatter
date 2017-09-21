package textformatter

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/mgutz/ansi"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh/terminal"
)

const defaultTimestampFormat = time.RFC3339Nano

type ColorScheme struct {
	Debug  string
	Info   string
	Warn   string
	Error  string
	Fatal  string
	Panic  string
	Tag    string
	Prefix string
	Func   string
}

type colorFunc func(string) string

type compiledColorScheme struct {
	Debug  colorFunc
	Info   colorFunc
	Warn   colorFunc
	Error  colorFunc
	Fatal  colorFunc
	Panic  colorFunc
	Tag    colorFunc
	Prefix colorFunc
	Func   colorFunc
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

func nocolor(v string) string {
	return v
}

var (
	baseTimestamp time.Time    = time.Now()
	defaultColors *ColorScheme = &ColorScheme{
		Debug:  "black+h",
		Info:   "white",
		Warn:   "yellow+h",
		Error:  "red+h",
		Fatal:  "red+h",
		Panic:  "red+h",
		Tag:    "magenta",
		Prefix: "cyan",
		Func:   "cyan+h",
	}
	noColors *compiledColorScheme = &compiledColorScheme{
		Debug:  nocolor,
		Info:   nocolor,
		Warn:   nocolor,
		Error:  nocolor,
		Fatal:  nocolor,
		Panic:  nocolor,
		Tag:    nocolor,
		Prefix: nocolor,
		Func:   nocolor,
	}
	defaultCompiledColorScheme *compiledColorScheme = compileColorScheme(defaultColors)
)

func miniTS() float64 {
	return time.Since(baseTimestamp).Seconds()
}

func getCompiledColor(main string, fallback string) colorFunc {
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
		Info:   getCompiledColor(s.Info, defaultColors.Info),
		Warn:   getCompiledColor(s.Warn, defaultColors.Warn),
		Error:  getCompiledColor(s.Error, defaultColors.Error),
		Fatal:  getCompiledColor(s.Fatal, defaultColors.Fatal),
		Panic:  getCompiledColor(s.Panic, defaultColors.Panic),
		Debug:  getCompiledColor(s.Debug, defaultColors.Debug),
		Tag:    getCompiledColor(s.Tag, defaultColors.Tag),
		Prefix: getCompiledColor(s.Prefix, defaultColors.Prefix),
		Func:   getCompiledColor(s.Func, defaultColors.Func),
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
	var levelColor colorFunc
	var levelText string
	switch entry.Level {
	case logrus.InfoLevel:
		levelColor = colors.Info
	case logrus.WarnLevel:
		levelColor = colors.Warn
	case logrus.ErrorLevel:
		levelColor = colors.Error
	case logrus.FatalLevel:
		levelColor = colors.Fatal
	case logrus.PanicLevel:
		levelColor = colors.Panic
	default:
		levelColor = colors.Debug
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
		fmt.Fprint(buf, levelColor(ts), " ")
	}

	fmt.Fprint(buf, levelColor(fmt.Sprintf("%5s", levelText)), " ")

	tv := "-"
	if v, ok := entry.Data["__t"]; ok {
		if tv, ok = v.(string); !ok {
			tv = fmt.Sprintf("%#v", v)
		}
	}
	fmt.Fprint(buf, colors.Tag(fmt.Sprintf("% 16s", tv)), " ")

	if v, ok := entry.Data["__p"]; ok {
		switch v.(type) {
		case string, fmt.Stringer:
			fmt.Fprint(buf, colors.Prefix(fmt.Sprintf("%s", v)))
		default:
			fmt.Fprint(buf, colors.Prefix(fmt.Sprintf("%T", v)))
		}
	} else {
		fmt.Fprint(buf, colors.Prefix("__p<missing>"))
	}

	if v, ok := entry.Data["__f"]; ok {
		fmt.Fprint(buf, " ", colors.Func(fmt.Sprintf("%v", v)))
	}
	fmt.Fprint(buf, "\t", levelColor(entry.Message), "\t")

	if v, ok := entry.Data[logrus.ErrorKey]; ok {
		printField(buf, logrus.ErrorKey, v, colors.Prefix, levelColor)
	}

	keys := make([]string, 0, len(entry.Data))
	for k := range entry.Data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		if k == "__p" || k == "__f" || k == "__t" || k == logrus.ErrorKey {
			continue
		}
		v := entry.Data[k]
		printField(buf, k, v, colors.Prefix, levelColor)
	}
	buf.WriteRune('\n')
	return buf.Bytes(), nil
}

func printField(buf *bytes.Buffer, k string, v interface{}, kcolor, vcolor colorFunc) {
	switch w := v.(type) {
	case fmt.Stringer:
		fmt.Fprintf(buf, " %s=%s", kcolor(k), vcolor(w.String()))
	case error:
		s := w.Error()
		r, n := utf8.DecodeRuneInString(s)
		fmt.Fprintf(buf, " %s=%s", kcolor(k), vcolor(fmt.Sprintf("%q", string(unicode.ToUpper(r))+s[n:]+"!")))
	default:
		fmt.Fprintf(buf, " %s=%s", kcolor(k), vcolor(fmt.Sprintf("%#v", v)))
	}
}
