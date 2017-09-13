package fmtfield

import "fmt"

type byteArray struct {
	v []byte
}

type format struct {
	f string
	a []interface{}
}

func ByteArrayHex(v []byte) fmt.Stringer {
	return &byteArray{v: v}
}

func Format(f string, a ...interface{}) fmt.Stringer {
	return &format{f: f, a: a}
}

func (f *byteArray) String() string {
	return fmt.Sprintf("[% X]", f.v)
}

func (f *format) String() string {
	return fmt.Sprintf(f.f, f.a...)
}
