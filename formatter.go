package textformatter

import (
	"bytes"
	"fmt"
	"github.com/mgutz/ansi"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh/terminal"
	"io"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
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

	PrefixFieldName  string
	PrefixFieldWidth int
	FuncFieldName    string
	TagFieldName     string
	TagFieldWidth    int

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
	// init
	f.Once.Do(func() {
		if len(f.PrefixFieldName) == 0 {
			f.PrefixFieldName = "__p"
		}
		if len(f.TagFieldName) == 0 {
			f.TagFieldName = "__t"
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
	var flen int

	// Tag
	if tf, ok := entry.Data[f.TagFieldName]; ok {
		switch t := tf.(type) {
		case fmt.Stringer:
			fstr = t.String()
		default:
			fstr = fmt.Sprintf("%#v", t)
		}
	} else {
		fstr = "-"
	}
	fmt.Fprint(buf, " ", f.colorScheme.Tag(fmt.Sprintf("%s", fstr)))
	flen = len(fstr)

	if flen < f.TagFieldWidth {
		fmt.Fprint(buf, strings.Repeat(" ", int(f.TagFieldWidth-flen)+1))
	} else {
		fmt.Fprint(buf, " ")
	}

	// Prefix
	if v, ok := entry.Data[f.PrefixFieldName]; ok {
		fstr = fmt.Sprintf("%v", v)
	} else {
		fstr = f.PrefixFieldName + "<missing>"
	}
	fmt.Fprint(buf, f.colorScheme.Prefix(fstr))
	flen = len(fstr)

	// Func
	if v, ok := entry.Data[f.FuncFieldName]; ok {
		fstr = fmt.Sprintf("%v", v)
		fmt.Fprint(buf, ".", f.colorScheme.Func(fstr))
		flen += len(fstr) + 1
	}

	if flen < f.PrefixFieldWidth {
		fmt.Fprint(buf, strings.Repeat(" ", int(f.PrefixFieldWidth-flen)+1))
	} else {
		fmt.Fprint(buf, " ")
	}

	// Message
	fmt.Fprint(buf, " ", levelColor(entry.Message))

	var errpresent bool
	if v, ok := entry.Data[logrus.ErrorKey]; ok {
		errpresent = true
		printField(buf, logrus.ErrorKey, v, f.colorScheme.Prefix, levelColor, true)
	}

	keys := make([]string, 0, len(entry.Data))
	for k := range entry.Data {
		switch k {
		case f.PrefixFieldName, f.TagFieldName, f.FuncFieldName, logrus.ErrorKey:
			continue
		default:
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	for n, k := range keys {
		v := entry.Data[k]
		printField(buf, k, v, f.colorScheme.Prefix, levelColor, n == 0 && !errpresent)
	}
	if errpresent || len(keys) > 0 {
		fmt.Fprint(buf, ")")
	}
	buf.WriteRune('\n')
	return buf.Bytes(), nil
}

func printField(buf *bytes.Buffer, k string, v interface{}, kcolor, vcolor colorFunc, first bool) {
	if first {
		fmt.Fprint(buf, " (")
	} else {
		fmt.Fprint(buf, " ")
	}
	switch w := v.(type) {
	case fmt.Stringer:
		fmt.Fprintf(buf, "%s=%s", kcolor(k), vcolor(w.String()))
	case error:
		fmt.Fprintf(buf, "%s={%s}", kcolor(k), vcolor(w.Error()))
	default:
		fmt.Fprintf(buf, "%s=%s", kcolor(k), vcolor(fmt.Sprintf("%#v", v)))
	}
}
