package field

import "fmt"

type formatter struct {
	f string
	v []interface{}
}

type singlevalueformatter struct {
	f string
	v interface{}
}

func (f *formatter) String() string {
	return fmt.Sprintf(f.f, f.v...)
}

func (f *singlevalueformatter) String() string {
	return fmt.Sprintf(f.f, f.v)
}

func Format(f string, a ...interface{}) fmt.Stringer {
	return &formatter{f: f, v: a}
}

func FormatHexArray(v []byte) fmt.Stringer {
	return &singlevalueformatter{f: "[% X]", v: v}
}

func FormatMoney(v uint64) fmt.Stringer {
	return &singlevalueformatter{f: "%d", v: v}
}
