// Package wrp provides a simple marshal/un-marshal interface for the wrp
// protocol
package wrp

import (
	"encoding/hex"
	"fmt"
	"github.com/ugorji/go/codec"
	"reflect"
)

const (
	AuthMsgType              = int64(1)
	SimpleReqResponseMsgType = int64(3)
	SimpleEventMsgType       = int64(4)
)

/*-- Authorization Message Type Handling -------------------------------------*/

type AuthStatusMsg struct {
	Status int64 `json:"status"   wrp:"status"`
}

func (m AuthStatusMsg) String() string {
	return fmt.Sprintf("SimpleReqResponseMsg{ Status: %d }\n", m.Status)
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

	m := tmp.(map[interface{}]interface{})
	msg_type := m["msg_type"]

	var v interface{}

	switch msg_type {
	case AuthMsgType:
		v = AuthStatusMsg{Status: m["status"].(int64)}
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
