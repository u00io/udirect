package forms

import (
	"encoding/binary"
	"errors"
)

func ParseString(bs []byte, offset int) (string, int, error) {
	if len(bs) < offset+4 {
		return "", 0, errors.New("parse_string_wrong_size_1")
	}
	size := binary.LittleEndian.Uint32(bs[offset : offset+4])
	offset += 4
	if len(bs) < offset+int(size) {
		return "", 0, errors.New("parse_string_wrong_size_2")
	}
	result := string(bs[offset : offset+int(size)])
	offset += int(size)
	return string(result), offset, nil
}

func SerializeString(str string) []byte {
	bs := make([]byte, 0)
	sizeOfStr := make([]byte, 4)
	binary.LittleEndian.PutUint32(sizeOfStr, uint32(len(str)))
	bs = append(bs, sizeOfStr...)
	bs = append(bs, []byte(str)...)
	return bs
}
