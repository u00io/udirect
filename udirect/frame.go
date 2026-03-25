package udirect

import (
	"encoding/binary"
	"errors"
)

/*
Format:
00000000 | 4 bytes = size of entire frame (including this bytes)
00000004 | 1 byte = frame type
00000005 | 3 bytes = reserved
00000008 | N bytes = frame payload (encrypted data or control message)
*/

var (
	ErrInvalidFrame = errors.New("invalid frame")
)

type frame struct {
	// Size uint32
	Type uint8
	// Reserved [3]byte
	Payload []byte
}

func newFrame(frameType uint8, payload []byte) *frame {
	var c frame
	c.Type = frameType
	c.Payload = payload
	return &c
}

func (c *frame) toBytes() []byte {
	size := 4 + 1 + 3 + len(c.Payload)
	buf := make([]byte, size)
	binary.LittleEndian.PutUint32(buf[0:4], uint32(size))
	buf[4] = c.Type
	copy(buf[8:], c.Payload)
	return buf
}

func (c *frame) toBytesInBuffer(buf []byte) int {
	size := 4 + 1 + 3 + len(c.Payload)
	if len(buf) < size {
		return 0
	}
	binary.LittleEndian.PutUint32(buf[0:4], uint32(size))
	buf[4] = c.Type
	copy(buf[8:], c.Payload)
	return size
}

func frameFromBytes(data []byte) (*frame, error) {
	if len(data) < 8 {
		return nil, ErrInvalidFrame
	}
	size := binary.LittleEndian.Uint32(data[0:4])
	if size != uint32(len(data)) {
		return nil, ErrInvalidFrame
	}
	frameType := data[4]
	payload := data[8:]
	return &frame{
		Type:    frameType,
		Payload: payload,
	}, nil
}
