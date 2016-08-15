// Package wrp provides a simple marshal/un-marshal interface for the wrp
// protocol
package wrp

import (
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/ugorji/go/codec"
	"io"
	"reflect"
)

const (
	AuthMsgType              = int64(2)
	SimpleReqResponseMsgType = int64(3)
	SimpleEventMsgType       = int64(4)
)

var (
	ErrorGetInt64    = errors.New("GetInt64 error casting value")
	ErrorInvalidType = errors.New("Invalid input type to wrp.Decode")
)

type WrpMsg interface {
	Origin() string
	Destination() string
}

// Encoder is implemented by any wrp message
type Encoder interface {
	Encode() ([]byte, error)
}

// writerTo is an internal type that adapts Encoder onto io.WriterTo
type writerTo struct {
	Encoder
}

func (w writerTo) WriteTo(output io.Writer) (int64, error) {
	data, err := w.Encode()
	if err != nil {
		return 0, err
	}

	count, err := output.Write(data)
	return int64(count), err
}

// WriterTo is a constructor function that produces an
// io.WriterTo from any Encoder.
func WriterTo(encoder Encoder) io.WriterTo {
	return writerTo{encoder}
}

/*-- Authorization Message Type Handling -------------------------------------*/

type AuthStatusMsg struct {
	Status int64 `json:"status"   wrp:"status"`
}

func (m AuthStatusMsg) String() string {
	return fmt.Sprintf("SimpleReqResponseMsg{ Status: %d }\n", m.Status)
}

func (m AuthStatusMsg) Origin() string {
	return ""
}

func (m AuthStatusMsg) Destination() string {
	return ""
}

/* Provide an encoder tied to the object type. */
func (m AuthStatusMsg) Encode() ([]byte, error) {
	return wrpEncode(AuthMsgType, m)
}

/*-- Simple Request/Response Message Type Handling ---------------------------*/

type SimpleReqResponseMsg struct {
	TransactionUUID string `json:"transaction_uuid" wrp:"transaction_uuid"`
	Source          string `json:"source"           wrp:"source"`
	Dest            string `json:"dest"             wrp:"dest"`
	Payload         []byte `json:"payload"          wrp:"payload"`
}

func (m SimpleReqResponseMsg) String() string {
	return fmt.Sprintf(
		"SimpleReqResponseMsg{\n"+
			"    TransactionUUID: '%s'\n"+
			"    Source:          '%s'\n"+
			"    Dest:            '%s'\n"+
			"    Payload:\n%s}\n",
		m.TransactionUUID,
		m.Source,
		m.Dest,
		hex.Dump(m.Payload))
}

func (m SimpleReqResponseMsg) Origin() string {
	return m.Source
}

func (m SimpleReqResponseMsg) Destination() string {
	return m.Dest
}

/* Provide an encoder tied to the object type. */
func (m SimpleReqResponseMsg) Encode() ([]byte, error) {
	return wrpEncode(SimpleReqResponseMsgType, m)
}

/*-- Simple Event Message Type Handling --------------------------------------*/
type SimpleEventMsg struct {
	Source  string `wrp:"source"`
	Dest    string `wrp:"dest"`
	Payload []byte `wrp:"payload"`
}

func (m SimpleEventMsg) String() string {
	return fmt.Sprintf(
		"SimpleReqResponseMsg{\n"+
			"    Source:  '%s'\n"+
			"    Dest:   '%s'\n"+
			"    Payload:\n%s}\n",
		m.Source,
		m.Dest,
		hex.Dump(m.Payload))
}

func (m SimpleEventMsg) Origin() string {
	return m.Source
}

func (m SimpleEventMsg) Destination() string {
	return m.Dest
}

/* Provide an encoder tied to the object type. */
func (m SimpleEventMsg) Encode() ([]byte, error) {
	return wrpEncode(SimpleEventMsgType, m)
}

/*-- The generic/global handlers follow --------------------------------------*/

/* This is the actual encoder that converts the wrp structure into
 * an array of bytes. */
func wrpEncode(mt int64, v interface{}) ([]byte, error) {

	st := reflect.TypeOf(v)
	m := map[string]interface{}{}

	m["msg_type"] = mt
	for i := 0; i < st.NumField(); i++ {
		tag := st.Field(i).Tag.Get("wrp")
		m[tag] = reflect.ValueOf(v).Field(i).Interface()
	}

	var buf []byte

	mh := new(codec.MsgpackHandle)
	mh.WriteExt = true
	mh.RawToString = true

	enc := codec.NewEncoderBytes(&buf, mh)

	if err := enc.Encode(m); nil != err {
		return nil, err
	}

	return buf, nil
}

// helper function to convert the different integer value types to
// int64; useful for scenarios where we don't know what type it is we're getting
func GetInt64(m map[interface{}]interface{}, key string) (int64, error) {
	switch valueType := m[key].(type) {
	case int8:
		return int64(valueType), nil
	case int16:
		return int64(valueType), nil
	case int32:
		return int64(valueType), nil
	case int64:
		return valueType, nil
	case int:
		return int64(valueType), nil
	case uint8:
		return int64(valueType), nil
	case uint16:
		return int64(valueType), nil
	case uint32:
		return int64(valueType), nil
	case uint64:
		return int64(valueType), nil
	case uint:
		return int64(valueType), nil
	default:
		return -1, ErrorGetInt64
	}
}

/* Decode the array of bytes into the right wrp structure. */
func Decode(buf []byte) (interface{}, error) {
	mh := new(codec.MsgpackHandle)
	mh.WriteExt = true
	mh.RawToString = true

	dec := codec.NewDecoderBytes(buf, mh)

	var tmp interface{}

	if err := dec.Decode(&tmp); nil != err {
		return nil, err
	}

	switch tmp.(type) {
	case map[interface{}]interface{}:
		// continue in the function
	default:
		return nil, ErrorInvalidType
	}

	m := tmp.(map[interface{}]interface{})
	msg_type := m["msg_type"]

	var v interface{}

	switch msg_type {
	case AuthMsgType:
		status, err := GetInt64(m, "status")
		if err != nil {
			return nil, fmt.Errorf("Error retrieving status: %v", err)
		}
		v = AuthStatusMsg{Status: status}
	case SimpleReqResponseMsgType:
		v = SimpleReqResponseMsg{Source: m["source"].(string),
			Dest:            m["dest"].(string),
			TransactionUUID: m["transaction_uuid"].(string),
			Payload:         m["payload"].([]byte)}
	case SimpleEventMsgType:
		v = SimpleEventMsg{Source: m["source"].(string),
			Dest:    m["dest"].(string),
			Payload: m["payload"].([]byte)}
	default:
		return nil, fmt.Errorf("Invalid 'msg_type': '%d'", msg_type)
	}

	return v, nil
}
