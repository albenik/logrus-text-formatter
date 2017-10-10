package textformatter

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

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

	PrefixFieldName  string
	PrefixFieldWidth int
	FuncFieldName    string

	colorScheme *compiledColorScheme

	sync.Once
}

func nocolor(v string) string {
	return v
}

var (
	baseTimestamp time.Time    = time.Now()
	defaultColors *ColorScheme = &ColorScheme{
		Debug:  "black+h",
		Info:   "green",
		Warn:   "yellow",
		Error:  "red",
		Fatal:  "red+h",
		Panic:  "red+h",
		Prefix: "cyan",
		Func:   "white",
	}
	noColors *compiledColorScheme = &compiledColorScheme{
		Debug:  nocolor,
		Info:   nocolor,
		Warn:   nocolor,
		Error:  nocolor,
		Fatal:  nocolor,
		Panic:  nocolor,
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
	// init
	f.Once.Do(func() {
		if len(f.PrefixFieldName) == 0 {
			f.PrefixFieldName = "__p"
		}
		if len(f.FuncFieldName) == 0 {
			f.FuncFieldName = "__f"
		}
		if len(f.TimestampFormat) == 0 {
			f.TimestampFormat = defaultTimestampFormat
		}
		if f.colorScheme == nil {
			if f.UseColors {
				f.colorScheme = defaultCompiledColorScheme
			} else {
				f.colorScheme = noColors
			}
		}
	})

	var buf *bytes.Buffer
	if entry.Buffer != nil {
		buf = entry.Buffer
	} else {
		buf = &bytes.Buffer{}
	}

	var levelColor colorFunc
	var levelText string
	switch entry.Level {
	case logrus.InfoLevel:
		levelColor = f.colorScheme.Info
	case logrus.WarnLevel:
		levelColor = f.colorScheme.Warn
	case logrus.ErrorLevel:
		levelColor = f.colorScheme.Error
	case logrus.FatalLevel:
		levelColor = f.colorScheme.Fatal
	case logrus.PanicLevel:
		levelColor = f.colorScheme.Panic
	default:
		levelColor = f.colorScheme.Debug
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
			ts = entry.Time.Format(f.TimestampFormat)
		}
		fmt.Fprint(buf, levelColor(ts), " ")
	}

	fmt.Fprint(buf, levelColor(fmt.Sprintf("%5s", levelText)))

	var fstr string

	// Prefix
	if v, ok := entry.Data[f.PrefixFieldName]; ok {
		fstr = fmt.Sprintf("%v", v)
	} else {
		fstr = f.PrefixFieldName + "<missing>"
	}
	flen := len(fstr)

	fmt.Fprint(buf, " ", f.colorScheme.Prefix(fstr))

	if flen < f.PrefixFieldWidth {
		fmt.Fprint(buf, strings.Repeat(" ", int(f.PrefixFieldWidth-flen)+1))
	} else {
		fmt.Fprint(buf, " ")
	}

	// Func
	if v, ok := entry.Data[f.FuncFieldName]; ok {
		fmt.Fprint(buf, " ", f.colorScheme.Func(fmt.Sprintf("%v", v)))
	}

	// Message
	fmt.Fprint(buf, " ", levelColor(entry.Message))

	var errpresent bool
	if v, ok := entry.Data[logrus.ErrorKey]; ok {
		errpresent = true
		printField(buf, logrus.ErrorKey, v, f.colorScheme.Func, levelColor, true)
	}

	keys := make([]string, 0, len(entry.Data))
	for k := range entry.Data {
		switch k {
		case f.PrefixFieldName, f.FuncFieldName, logrus.ErrorKey:
			continue
		default:
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	for n, k := range keys {
		v := entry.Data[k]
		printField(buf, k, v, f.colorScheme.Func, levelColor, n == 0 && !errpresent)
	}
	if errpresent || len(keys) > 0 {
		fmt.Fprint(buf, ")")
	}
	fmt.Fprint(buf, "\n")

	return buf.Bytes(), nil
}

func printField(w io.Writer, key string, val interface{}, kcolor, vcolor colorFunc, first bool) {
	if first {
		fmt.Fprint(w, " (")
	} else {
		fmt.Fprint(w, " ")
	}
	switch v := val.(type) {
	case fmt.Stringer:
		fmt.Fprintf(w, "%s=%s", kcolor(key), vcolor(v.String()))
	case error:
		fmt.Fprintf(w, "%s={%s}", kcolor(key), vcolor(v.Error()))
	default:
		fmt.Fprintf(w, "%s=%s", kcolor(key), vcolor(fmt.Sprintf("%#v", v)))
	}
}
