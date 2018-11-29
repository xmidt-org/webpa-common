package wrp

import (
	"errors"
)

// Constant HTTP header strings representing WRP fields
const (
	MsgTypeHeader         = "X-Midt-Msg-Type"
	TransactionUuidHeader = "X-Midt-Transaction-Uuid"
	StatusHeader          = "X-Midt-Status"
	RDRHeader             = "X-Midt-Request-Delivery-Response"
	HeadersArrHeader      = "X-Midt-Headers"
	IncludeSpansHeader    = "X-Midt-Include-Spans"
	SpansHeader           = "X-Midt-Spans"
	PathHeader            = "X-Midt-Path"
	SourceHeader          = "X-Midt-Source"
	MoneyTraceHeader      = "X-Money-Trace"
	MoneySpanHeader       = "X-Money-Span"
)

var ErrInvalidMsgType = errors.New("Invalid Message Type")

// Map string to MessageType int
/*
func StringToMessageType(str string) MessageType {
	switch str {
	case "Auth":
		return AuthMessageType
	case "SimpleRequestResponse":
		return SimpleRequestResponseMessageType
	case "SimpleEvent":
		return SimpleEventMessageType
	case "Create":
		return CreateMessageType
	case "Retrieve":
		return RetrieveMessageType
	case "Update":
		return UpdateMessageType
	case "Delete":
		return DeleteMessageType
	case "ServiceRegistration":
		return ServiceRegistrationMessageType
	case "ServiceAlive":
		return ServiceAliveMessageType
	default:
		return -1
	}
}
*/

// Convert HTTP header to WRP generic Message
/*
func HeaderToWRP(header http.Header) (*Message, error) {
	msg := new(Message)

	// MessageType is mandatory
	msgType := header.Get(MsgTypeHeader)
	if !strings.EqualFold(msgType, "") && StringToMessageType(msgType) != MessageType(-1) {
		msg.Type = StringToMessageType(msgType)
	} else {
		return nil, ErrInvalidMsgType
	}

	if src := header.Get(SourceHeader); !strings.EqualFold(src, "") {
		msg.Source = src
	}

	if transUuid := header.Get(TransactionUuidHeader); !strings.EqualFold(transUuid, "") {
		msg.TransactionUUID = transUuid
	}

	if status := header.Get(StatusHeader); !strings.EqualFold(status, "") {
		if statusInt, err := strconv.ParseInt(status, 10, 64); err == nil {
			msg.SetStatus(statusInt)
		} else {
			return nil, err
		}
	}

	if rdr := header.Get(RDRHeader); !strings.EqualFold(rdr, "") {
		if rdrInt, err := strconv.ParseInt(rdr, 10, 64); err == nil {
			msg.SetRequestDeliveryResponse(rdrInt)
		} else {
			return nil, err
		}
	}

	if path := header.Get(PathHeader); !strings.EqualFold(path, "") {
		msg.Path = path
	}

	if includeSpans := header.Get(IncludeSpansHeader); !strings.EqualFold(includeSpans, "") {
		if spansBool, err := strconv.ParseBool(includeSpans); err == nil {
			msg.SetIncludeSpans(spansBool)
		} else {
			return nil, err
		}
	}

	// Handle Headers and Spans which contain multiple values
	for key, value := range header {
		if strings.EqualFold(key, HeadersArrHeader) {
			if msg.Headers == nil {
				msg.Headers = []string{}
			}
			for item := range value {
				msg.Headers = append(msg.Headers, value[item])
			}
		}

		// Each span element will look like this {"name" , "start_time" , "duration"}
		if strings.EqualFold(key, SpansHeader) {
			if msg.Spans == nil {
				msg.Spans = make([][]string, len(value))
			}

			j := 0
			for i := 0; i < len(value); i++ {
				msg.Spans[j] = append(msg.Spans[j], value[i])
				if (i+1)%3 == 0 {
					j++
				}
			}
		}
	}

	return msg, nil
}
*/
// Convert WRP generic Message to HTTP header
/*
func WRPToHeader(msg *Message) (header http.Header, err error) {

	header = make(map[string][]string)

	// Message Type is mandatory
	if strings.EqualFold(msg.Type.String(), InvalidMessageTypeString) {
		return nil, ErrInvalidMsgType
	} else {
		header.Add(MsgTypeHeader, msg.Type.String())
	}

	if msg.Status != nil {
		header.Add(StatusHeader, strconv.FormatInt(*msg.Status, 10))
	}

	if !strings.EqualFold(msg.Source, "") {
		header.Add(SourceHeader, msg.Source)
	}

	if !strings.EqualFold(msg.TransactionUUID, "") {
		header.Add(TransactionUuidHeader, msg.TransactionUUID)
	}

	if !strings.EqualFold(msg.Path, "") {
		header.Add(PathHeader, msg.Path)
	}

	if msg.RequestDeliveryResponse != nil {
		header.Add(RDRHeader, strconv.FormatInt(*msg.RequestDeliveryResponse, 10))
	}

	if msg.IncludeSpans != nil {
		header.Add(IncludeSpansHeader, strconv.FormatBool(*msg.IncludeSpans))
	}

	if msg.Spans != nil {
		for i := 0; i < len(msg.Spans); i++ {
			for _, span := range msg.Spans[i] {
				header.Add(SpansHeader, span)
			}
		}
	}

	if msg.Headers != nil {
		if msg.Headers != nil {
			for _, hdr := range msg.Headers {
				header.Add(HeadersArrHeader, hdr)
			}
		}
	}

	return
}
*/
