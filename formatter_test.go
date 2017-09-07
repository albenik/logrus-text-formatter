package textformatter_test

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/albenik/logrus-text-formatter"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestInstance_Format(t *testing.T) {
	now := time.Now()
	f := &textformatter.Instance{ForceFormatting: true, FullTimestamp: true, DisableColors: true}
	t.Run("Simple", func(t *testing.T) {
		out := bytes.NewBuffer(nil)
		entry := &logrus.Entry{
			Logger:  nil,
			Buffer:  out,
			Level:   logrus.DebugLevel,
			Time:    now,
			Message: "TeSt",
		}
		s, err := f.Format(entry)
		assert.NoError(t, err)
		assert.Equal(t, fmt.Sprintf("%s DEBUG __p<missing>: TeSt\n", now.Format(time.RFC3339Nano)), string(s))
	})
	t.Run("Only __t", func(t *testing.T) {
		out := bytes.NewBuffer(nil)
		entry := &logrus.Entry{
			Logger:  nil,
			Buffer:  out,
			Level:   logrus.DebugLevel,
			Time:    now,
			Message: "TeSt",
			Data:    logrus.Fields{"__t": "12345"},
		}
		s, err := f.Format(entry)
		assert.NoError(t, err)
		assert.Equal(t, fmt.Sprintf("%s DEBUG :12345: __p<missing>: TeSt\n", now.Format(time.RFC3339Nano)), string(s))
	})
	t.Run("Only __p", func(t *testing.T) {
		out := bytes.NewBuffer(nil)
		entry := &logrus.Entry{
			Logger:  nil,
			Buffer:  out,
			Level:   logrus.DebugLevel,
			Time:    now,
			Message: "TeSt",
			Data:    logrus.Fields{"__p": "ppp"},
		}
		s, err := f.Format(entry)
		assert.NoError(t, err)
		assert.Equal(t, fmt.Sprintf("%s DEBUG ppp: TeSt\n", now.Format(time.RFC3339Nano)), string(s))
	})
	t.Run("Only __f", func(t *testing.T) {
		out := bytes.NewBuffer(nil)
		entry := &logrus.Entry{
			Logger:  nil,
			Buffer:  out,
			Level:   logrus.DebugLevel,
			Time:    now,
			Message: "TeSt",
			Data:    logrus.Fields{"__f": "fff"},
		}
		s, err := f.Format(entry)
		assert.NoError(t, err)
		assert.Equal(t, fmt.Sprintf("%s DEBUG __p<missing>.fff: TeSt\n", now.Format(time.RFC3339Nano)), string(s))
	})
	t.Run("Combined __t & __p", func(t *testing.T) {
		out := bytes.NewBuffer(nil)
		entry := &logrus.Entry{
			Logger:  nil,
			Buffer:  out,
			Level:   logrus.DebugLevel,
			Time:    now,
			Message: "TeSt",
			Data:    logrus.Fields{"__t": "12345", "__p": "ppp"},
		}
		s, err := f.Format(entry)
		assert.NoError(t, err)
		assert.Equal(t, fmt.Sprintf("%s DEBUG :12345: ppp: TeSt\n", now.Format(time.RFC3339Nano)), string(s))
	})
	t.Run("Combined __t & __f", func(t *testing.T) {
		out := bytes.NewBuffer(nil)
		entry := &logrus.Entry{
			Logger:  nil,
			Buffer:  out,
			Level:   logrus.DebugLevel,
			Time:    now,
			Message: "TeSt",
			Data:    logrus.Fields{"__t": "12345", "__f": "fff"},
		}
		s, err := f.Format(entry)
		assert.NoError(t, err)
		assert.Equal(t, fmt.Sprintf("%s DEBUG :12345: __p<missing>.fff: TeSt\n", now.Format(time.RFC3339Nano)), string(s))
	})
	t.Run("Combined __p & __f", func(t *testing.T) {
		out := bytes.NewBuffer(nil)
		entry := &logrus.Entry{
			Logger:  nil,
			Buffer:  out,
			Level:   logrus.DebugLevel,
			Time:    now,
			Message: "TeSt",
			Data:    logrus.Fields{"__p": "ppp", "__f": "fff"},
		}
		s, err := f.Format(entry)
		assert.NoError(t, err)
		assert.Equal(t, fmt.Sprintf("%s DEBUG ppp.fff: TeSt\n", now.Format(time.RFC3339Nano)), string(s))
	})
	t.Run("Combined __t & __p & __f", func(t *testing.T) {
		out := bytes.NewBuffer(nil)
		entry := &logrus.Entry{
			Logger:  nil,
			Buffer:  out,
			Level:   logrus.DebugLevel,
			Time:    now,
			Message: "TeSt",
			Data:    logrus.Fields{"__t": "12345", "__p": "ppp", "__f": "fff"},
		}
		s, err := f.Format(entry)
		assert.NoError(t, err)
		assert.Equal(t, fmt.Sprintf("%s DEBUG :12345: ppp.fff: TeSt\n", now.Format(time.RFC3339Nano)), string(s))
	})
}
