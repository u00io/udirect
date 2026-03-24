package forms

import (
	"encoding/binary"
	"errors"
)

func ParseBytes(bs []byte, offset int) ([]byte, int, error) {
	if len(bs) < offset+4 {
		return nil, 0, errors.New("parse_string_wrong_size_1")
	}
	size := binary.LittleEndian.Uint32(bs[offset : offset+4])
	offset += 4
	if len(bs) < offset+int(size) {
		return nil, 0, errors.New("parse_string_wrong_size_2")
	}
	result := bs[offset : offset+int(size)]
	offset += int(size)
	return result, offset, nil
}

func ParseString(bs []byte, offset int) (string, int, error) {
	raw, nextOffset, err := ParseBytes(bs, offset)
	if err != nil {
		return "", 0, err
	}
	return string(raw), nextOffset, nil
}

func SerializeString(str string) []byte {
	return SerializeBytes([]byte(str))
}

func SerializeBytes(v []byte) []byte {
	bs := make([]byte, 4+len(v))
	binary.LittleEndian.PutUint32(bs[0:4], uint32(len(v)))
	copy(bs[4:], v)
	return bs
}
