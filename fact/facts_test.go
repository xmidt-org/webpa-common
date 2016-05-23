package fact

import (
	"encoding/base64"
	"github.com/Comcast/webpa-common/canonical"
	"github.com/Comcast/webpa-common/convey"
	"github.com/Comcast/webpa-common/logging"
	"golang.org/x/net/context"
	"os"
	"reflect"
	"testing"
)

const (
	conveyPayload string = "eyAicGFyYW1ldGVycyI6IFsgeyAibmFtZSI6ICJEZXZpY2UuRGV2aWNlSW5mby5XZWJwYS5YX0NPTUNBU1QtQ09NX0NJRCIsICJ2YWx1ZSI6ICIwIiwgImRhdGFUeXBlIjogMCB9LCB7ICJuYW1lIjogIkRldmljZS5EZXZpY2VJbmZvLldlYnBhLlhfQ09NQ0FTVC1DT01fQ01DIiwgInZhbHVlIjogIjI2OSIsICJkYXRhVHlwZSI6IDIgfSBdIH0K"
)

func TestLogger(t *testing.T) {
	ctx := context.Background()
	if value, ok := Logger(ctx); value != nil {
		t.Error("Logger() must return nil when no logger is present")
	} else if ok {
		t.Error("Logger() must return false when no logger is present")
	}

	func() {
		defer func() {
			if recovered := recover(); recovered == nil {
				t.Error("MustLogger() must panic when no logger is present")
			} else if recovered != NoLogger {
				t.Errorf("MustLogger() must panic with [%v] when no logger is present", NoLogger)
			}
		}()

		MustLogger(ctx)
	}()

	logger := &logging.LoggerWriter{os.Stdout}
	ctx = SetLogger(ctx, logger)
	if value, ok := Logger(ctx); value != logger {
		t.Error("Logger() must return the previously set value")
	} else if !ok {
		t.Error("Logger() must return true when a logger is present")
	}

	if MustLogger(ctx) != logger {
		t.Error("MustLogger() must return the previously set value")
	}
}

func TestDeviceId(t *testing.T) {
	ctx := context.Background()
	if value, ok := DeviceId(ctx); value != nil {
		t.Error("DeviceId() must return nil when no device id is present")
	} else if ok {
		t.Error("DeviceId() must return false when no device id is present")
	}

	func() {
		defer func() {
			if recovered := recover(); recovered == nil {
				t.Error("MustDeviceId() must panic when no device id is present")
			} else if recovered != NoDeviceId {
				t.Errorf("MustDeviceId() must panic with [%v] when no device id is present", NoLogger)
			}
		}()

		MustDeviceId(ctx)
	}()

	deviceId, err := canonical.ParseId("mac:111122223333")
	if err != nil {
		t.Fatalf("Could not parse device id: %v", err)
	}

	t.Logf("Parsed device id: %v", deviceId)
	ctx = SetDeviceId(ctx, deviceId)
	if value, ok := DeviceId(ctx); value != deviceId {
		t.Error("DeviceId() must return the previously set value")
	} else if !ok {
		t.Error("DeviceId() must return true when a device id is present")
	}

	if MustDeviceId(ctx) != deviceId {
		t.Error("MustDeviceId() must return the previously set value")
	}
}

func TestConvey(t *testing.T) {
	ctx := context.Background()
	if value, ok := Convey(ctx); value != nil {
		t.Error("Convey() must return nil when no convey payload is present")
	} else if ok {
		t.Error("Convey() must return false when no convey payload is present")
	}

	func() {
		defer func() {
			if recovered := recover(); recovered == nil {
				t.Error("MustConvey() must panic when no convey payload is present")
			} else if recovered != NoConvey {
				t.Errorf("MustConvey() must panic with [%v] when no convey payload is present", NoConvey)
			}
		}()

		MustConvey(ctx)
	}()

	payload, err := convey.ParsePayload(base64.StdEncoding, conveyPayload)
	if err != nil {
		t.Fatalf("Could not parse convey payload: %v", err)
	}

	t.Logf("Parsed payload: %v", payload)
	ctx = SetConvey(ctx, payload)
	if value, ok := Convey(ctx); !reflect.DeepEqual(value, payload) {
		t.Error("Convey() must return the previously set value")
	} else if !ok {
		t.Error("Convey() must return true when a convey payload is present")
	}

	if !reflect.DeepEqual(MustConvey(ctx), payload) {
		t.Error("MustConvey() must return the previously set value")
	}
}
