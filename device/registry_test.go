package device

import (
	"testing"
	"time"

	"github.com/Comcast/webpa-common/logging"
	"github.com/stretchr/testify/assert"
)

var (
	connectedAt  = time.Now()
	nosuchID     = ID("nosuch ID")
	nosuchDevice = newDevice(nosuchID, 1, connectedAt, logging.DefaultLogger())

	singleID     = ID("single")
	singleDevice = newDevice(singleID, 1, connectedAt, logging.DefaultLogger())

	doubleID      = ID("double")
	doubleDevice1 = newDevice(doubleID, 1, connectedAt, logging.DefaultLogger())
	doubleDevice2 = newDevice(doubleID, 1, connectedAt, logging.DefaultLogger())

	manyID      = ID("many")
	manyDevice1 = newDevice(manyID, 1, connectedAt, logging.DefaultLogger())
	manyDevice2 = newDevice(manyID, 1, connectedAt, logging.DefaultLogger())
	manyDevice3 = newDevice(manyID, 1, connectedAt, logging.DefaultLogger())
	manyDevice4 = newDevice(manyID, 1, connectedAt, logging.DefaultLogger())
	manyDevice5 = newDevice(manyID, 1, connectedAt, logging.DefaultLogger())
)

func testRegistry(t *testing.T, assert *assert.Assertions) *registry {
	registry := newRegistry(1000)
	if !assert.NotNil(registry) {
		t.FailNow()
	}

	assert.Nil(registry.add(singleDevice))
	assert.Nil(registry.add(doubleDevice1))
	assert.Nil(registry.add(doubleDevice2))
	assert.Nil(registry.add(manyDevice1))
	assert.Nil(registry.add(manyDevice2))
	assert.Nil(registry.add(manyDevice3))
	assert.Nil(registry.add(manyDevice4))
	assert.Nil(registry.add(manyDevice5))

	return registry
}

func TestRegistryDuplicateDevice(t *testing.T) {
	assert := assert.New(t)
	registry := testRegistry(t, assert)

	duplicateDevice := newDevice(ID("duplicate device"), 1, time.Now(), logging.DefaultLogger())
	assert.Nil(registry.add(duplicateDevice))
	assert.Equal(ErrorDuplicateDevice, registry.add(duplicateDevice))
}

func TestRegistryVisitID(t *testing.T) {
	assert := assert.New(t)
	testData := []struct {
		expectedID    ID
		expectVisited deviceSet
	}{
		{nosuchID, expectsDevices()},
		{singleID, expectsDevices(singleDevice)},
		{doubleID, expectsDevices(doubleDevice1, doubleDevice2)},
		{manyID, expectsDevices(manyDevice1, manyDevice2, manyDevice3, manyDevice4, manyDevice5)},
	}

	for _, record := range testData {
		t.Logf("%#v", record)
		registry := testRegistry(t, assert)
		actualVisited := deviceSet{}

		assert.Equal(
			len(record.expectVisited),
			registry.visitID(record.expectedID, actualVisited.registryCapture()),
		)

		assert.Equal(record.expectVisited, actualVisited)
	}
}

func TestRegistryVisitIf(t *testing.T) {
	assert := assert.New(t)
	testData := []struct {
		filter        func(ID) bool
		expectVisited deviceSet
	}{
		{func(ID) bool { return false }, expectsDevices()},
		{func(id ID) bool { return id == singleID }, expectsDevices(singleDevice)},
		{func(id ID) bool { return id == doubleID }, expectsDevices(doubleDevice1, doubleDevice2)},
		{func(id ID) bool { return id == manyID }, expectsDevices(manyDevice1, manyDevice2, manyDevice3, manyDevice4, manyDevice5)},
	}

	for _, record := range testData {
		t.Logf("%#v", record)
		registry := testRegistry(t, assert)
		actualVisited := deviceSet{}

		assert.Equal(
			len(record.expectVisited),
			registry.visitIf(record.filter, actualVisited.registryCapture()),
		)

		assert.Equal(record.expectVisited, actualVisited)
	}
}

func TestRegistryVisitAll(t *testing.T) {
	assert := assert.New(t)
	registry := testRegistry(t, assert)

	expectVisited := expectsDevices(singleDevice, doubleDevice1, doubleDevice2, manyDevice1, manyDevice2, manyDevice3, manyDevice4, manyDevice5)

	actualVisited := deviceSet{}
	assert.Equal(len(expectVisited), registry.visitAll(actualVisited.registryCapture()))
	assert.Equal(expectVisited, actualVisited)
}

func TestRegistryRemoveAll(t *testing.T) {
	assert := assert.New(t)
	testData := []struct {
		idToRemove     ID
		expectRemoved  deviceSet
		expectVisitAll deviceSet
	}{
		{
			nosuchID,
			expectsDevices(),
			expectsDevices(singleDevice, doubleDevice1, doubleDevice2, manyDevice1, manyDevice2, manyDevice3, manyDevice4, manyDevice5),
		},
		{
			singleID,
			expectsDevices(singleDevice),
			expectsDevices(doubleDevice1, doubleDevice2, manyDevice1, manyDevice2, manyDevice3, manyDevice4, manyDevice5),
		},
		{
			doubleID,
			expectsDevices(doubleDevice1, doubleDevice2),
			expectsDevices(singleDevice, manyDevice1, manyDevice2, manyDevice3, manyDevice4, manyDevice5),
		},
		{
			manyID,
			expectsDevices(manyDevice1, manyDevice2, manyDevice3, manyDevice4, manyDevice5),
			expectsDevices(singleDevice, doubleDevice1, doubleDevice2),
		},
	}

	for _, record := range testData {
		t.Logf("%v", record)
		registry := testRegistry(t, assert)

		removed := registry.removeAll(record.idToRemove)
		assert.Equal(len(record.expectRemoved), len(removed))
		for _, d := range removed {
			assert.True(record.expectRemoved[d])
		}

		actualVisitAll := make(deviceSet)
		registry.visitAll(actualVisitAll.registryCapture())
		assert.Equal(record.expectVisitAll, actualVisitAll)
	}
}
