package fieldfmt

import "fmt"

type byteArray struct {
	v []byte
}

func ByteArray(v []byte) fmt.Stringer {
	return &byteArray{v: v}
}

func (f *byteArray) String() string {
	return fmt.Sprintf("[% X]", f.v)
}
