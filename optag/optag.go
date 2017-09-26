package optag

import (
	"fmt"
	"strconv"
	"time"
)

type Tag interface {
	fmt.Stringer
	Time() time.Time
}

type tag struct {
	time time.Time
	str  string
}

func (t *tag) Time() time.Time {
	return t.time
}

func (t *tag) String() string {
	return t.str
}

func New(parent Tag) Tag {
	now := time.Now()
	if parent == nil {
		return &tag{
			time: now,
			str:  strconv.FormatInt(time.Now().UnixNano(), 36),
		}
	} else {
		return &tag{
			time: now,
			str:  parent.String() + " " + strconv.FormatInt(now.Sub(parent.Time()).Nanoseconds(), 36),
		}
	}
}
