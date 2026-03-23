package forms

import (
	"encoding/binary"
	"fmt"
	"math"
	"time"
)

type ByteArray []byte

type Form struct {
	Items map[string]ByteArray
	Types map[string]byte
}

const (
	TYPE_VECTORMASK = 0x80
	TYPE_BYTES      = 0x00
	TYPE_BOOL       = 0x01
	TYPE_INT64      = 0x02
	TYPE_DOUBLE     = 0x03
	TYPE_STRING     = 0x04
	TYPE_FORM       = 0x05
)

func NewForm() *Form {
	var c Form
	c.Items = make(map[string]ByteArray)
	c.Types = make(map[string]byte)
	return &c
}

func (c *Form) HasField(name string) bool {
	_, ok := c.Items[name]
	return ok
}

func (c *Form) SetField(name string, value ByteArray) {
	c.Items[name] = value
}

func (c *Form) GetField(name string) ByteArray {
	if _, ok := c.Items[name]; !ok {
		return nil
	}

	return c.Items[name]
}

func (c *Form) SetFieldDateTime(name string, value time.Time) {
	msec := value.UnixNano() / int64(time.Millisecond)
	c.SetFieldInt64(name, msec)
}

func (c *Form) GetFieldDateTime(name string) time.Time {
	msec := c.GetFieldInt64(name)
	if msec == 0 {
		return time.Time{}
	}
	return time.Unix(0, msec*int64(time.Millisecond)).UTC()
}

func (c *Form) GetFieldString(name string) string {
	if _, ok := c.Items[name]; !ok {
		return ""
	}

	return string(c.Items[name])
}

func (c *Form) SetFieldString(name string, value string) {
	c.Items[name] = ByteArray(value)
	c.Types[name] = TYPE_STRING
}

func (c *Form) GetFieldInt64(name string) int64 {
	if v, ok := c.Items[name]; ok {
		if len(v) < 8 {
			return 0
		}
		value := binary.LittleEndian.Uint64(v)
		return int64(value)
	}
	return 0
}

func (c *Form) SetFieldInt64(name string, value int64) {
	bs := make([]byte, 8)
	binary.LittleEndian.PutUint64(bs, uint64(value))
	c.Items[name] = bs
	c.Types[name] = TYPE_INT64
}

func (c *Form) GetFieldForm(name string) *Form {
	if _, ok := c.Items[name]; !ok {
		return NewForm()
	}
	form, err := ParseForm(c.Items[name])
	if err != nil {
		return NewForm()
	}
	return form
}

func (c *Form) SetFieldForm(name string, form *Form) {
	bs := form.Serialize()
	c.SetField(name, bs)
	c.Types[name] = TYPE_FORM
}

func (c *Form) SetFieldVectorString(name string, values []string) {
	form := NewForm()
	form.SetFieldInt64("count", int64(len(values)))
	for i, str := range values {
		form.SetFieldString(fmt.Sprint(i), str)
	}
	bs := form.Serialize()
	c.SetField(name, bs)
	c.Types[name] = TYPE_VECTORMASK | TYPE_STRING
}

func (c *Form) GetFieldVectorString(name string) []string {
	if _, ok := c.Items[name]; !ok {
		return nil
	}

	form, err := ParseForm(c.Items[name])
	if err != nil {
		return nil
	}
	var result []string
	for i := int64(0); i < form.GetFieldInt64("count"); i++ {
		result = append(result, form.GetFieldString(fmt.Sprint(i)))
	}
	return result
}

func (c *Form) GetFieldVectorFloat64(name string) []float64 {
	if _, ok := c.Items[name]; !ok {
		return nil
	}

	data := c.GetField(name)
	result := make([]float64, len(data)/8)
	for i := 0; i < len(result); i++ {
		result[i] = math.Float64frombits(binary.LittleEndian.Uint64(data[i*8 : (i+1)*8]))
	}
	return result
}

func (c *Form) SetFieldVectorFloat64(name string, values []float64) {
	data := make([]byte, len(values)*8)
	for i, value := range values {
		binary.LittleEndian.PutUint64(data[i*8:(i+1)*8], math.Float64bits(value))
	}
	c.SetField(name, data)
	c.Types[name] = TYPE_VECTORMASK | TYPE_DOUBLE
}

func (c *Form) GetFieldVectorInt64(name string) []int64 {
	if _, ok := c.Items[name]; !ok {
		return nil
	}

	data := c.GetField(name)
	result := make([]int64, len(data)/8)
	for i := 0; i < len(result); i++ {
		result[i] = int64(binary.LittleEndian.Uint64(data[i*8 : (i+1)*8]))
	}
	return result
}

func (c *Form) SetFieldVectorInt64(name string, values []int64) {
	data := make([]byte, len(values)*8)
	for i, value := range values {
		binary.LittleEndian.PutUint64(data[i*8:(i+1)*8], uint64(value))
	}
	c.SetField(name, data)
	c.Types[name] = TYPE_VECTORMASK | TYPE_INT64
}

func (c *Form) SetFieldVectorForms(name string, forms []*Form) {
	form := NewForm()
	form.SetFieldInt64("count", int64(len(forms)))
	for i, subForm := range forms {
		bs := subForm.Serialize()
		form.SetField(fmt.Sprint(i), bs)
	}
	c.SetField(name, form.Serialize())
	c.Types[name] = TYPE_VECTORMASK | TYPE_FORM
}

func (c *Form) GetFieldVectorForms(name string) []*Form {
	if _, ok := c.Items[name]; !ok {
		return nil
	}
	form, err := ParseForm(c.Items[name])
	if err != nil {
		return nil
	}
	var result []*Form
	for i := int64(0); i < form.GetFieldInt64("count"); i++ {
		subFormData := form.GetField(fmt.Sprint(i))
		subForm, err := ParseForm(subFormData)
		if err != nil {
			continue
		}
		result = append(result, subForm)
	}
	return result
}

func (c *Form) SetFieldBool(name string, value bool) {
	bs := make([]byte, 1)
	if value {
		bs[0] = 1
	} else {
		bs[0] = 0
	}
	c.SetField(name, bs)
}

func (c *Form) GetFieldBool(name string) bool {
	bs := c.GetField(name)
	if len(bs) < 1 {
		return false
	}
	return bs[0] != 0
}

func (c *Form) SetFieldDouble(name string, value float64) {
	bs := make([]byte, 8)
	binary.LittleEndian.PutUint64(bs, math.Float64bits(value))
	c.SetField(name, bs)
	c.Types[name] = TYPE_DOUBLE
}

func (c *Form) GetFieldDouble(name string) float64 {
	bs := c.GetField(name)
	if len(bs) < 8 {
		return 0.0
	}
	return math.Float64frombits(binary.LittleEndian.Uint64(bs))
}

func (c *Form) Serialize() []byte {
	bs := make([]byte, 0)
	for key, value := range c.Items {
		bs = append(bs, SerializeString(key)...)

		tp := byte(0)
		if v, ok := c.Types[key]; ok {
			tp = v
		} else {
			tp = 0 // default type
		}

		bs = append(bs, tp)
		bs = append(bs, SerializeString(string(value))...)
	}
	return bs
}
