package sessionid

import (
	"encoding/base64"
	"encoding/binary"
	"errors"
	"math/rand"
	"time"
)

func GenerateID() string {
	return GenerateIDWithTime(time.Now())
}

func GenerateIDWithTime(t time.Time) string {
	var buffer [16]byte
	rand.Read(buffer[:])
	ts := uint32(t.Unix())
	binary.BigEndian.PutUint32(buffer[:4], ts)
	return base64.RawURLEncoding.EncodeToString(buffer[:])
}

func ParseID(id string) (time.Time, error) {
	buffer, err := base64.RawURLEncoding.DecodeString(id)
	if err != nil {
		return time.Time{}, err
	}
	if len(buffer) != 16 {
		return time.Time{}, errors.New("byte array is wrong length")
	}
	ts := binary.BigEndian.Uint32(buffer[:4])
	return time.Unix(int64(ts), 0), nil
}
