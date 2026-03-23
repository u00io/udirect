package form

import "errors"

func ParseForm(bs []byte) (*Form, error) {
	var err error
	c := NewForm()
	offset := 0
	for offset < len(bs) {
		var name string
		var value string
		name, offset, err = ParseString(bs, offset)
		if err != nil {
			return nil, err
		}
		// Read type
		if len(bs) < offset+1 {
			return nil, errors.New("parse_form_wrong_size_1")
		}
		tp := bs[offset]
		c.Types[name] = tp
		offset++
		value, offset, err = ParseString(bs, offset)
		if err != nil {
			return nil, err
		}
		c.Items[name] = ByteArray(value)
	}
	return c, nil
}
