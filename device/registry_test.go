package device

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

// visitedDevices is a convenient map type for capturing the devices visited within a registry
type visitedDevices map[*device]bool

func (v visitedDevices) capture() func(*device) {
	return func(d *device) {
		v[d] = true
	}
}

func expectsVisited(devices ...*device) visitedDevices {
	visited := make(visitedDevices, len(devices))
	for _, d := range devices {
		visited[d] = true
	}

	return visited
}

var (
	nosuchID     = ID("nosuch ID")
	nosuchKey    = Key("nosuch key")
	nosuchDevice = newDevice(nosuchID, nosuchKey, nil, 1)

	singleID     = ID("single")
	singleKey    = Key("single key")
	singleDevice = newDevice(singleID, singleKey, nil, 1)

	doubleID      = ID("double")
	doubleKey1    = Key("double key 1")
	doubleDevice1 = newDevice(doubleID, doubleKey1, nil, 1)
	doubleKey2    = Key("double key 2")
	doubleDevice2 = newDevice(doubleID, doubleKey2, nil, 1)

	manyID      = ID("many")
	manyKey1    = Key("many key 1")
	manyDevice1 = newDevice(manyID, manyKey1, nil, 1)
	manyKey2    = Key("many key 2")
	manyDevice2 = newDevice(manyID, manyKey2, nil, 1)
	manyKey3    = Key("many key 3")
	manyDevice3 = newDevice(manyID, manyKey3, nil, 1)
	manyKey4    = Key("many key 4")
	manyDevice4 = newDevice(manyID, manyKey4, nil, 1)
	manyKey5    = Key("many key 5")
	manyDevice5 = newDevice(manyID, manyKey5, nil, 1)
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

func TestRegistryVisitID(t *testing.T) {
	assert := assert.New(t)
	testData := []struct {
		expectedID    ID
		expectVisited visitedDevices
	}{
		{nosuchID, expectsVisited()},
		{singleID, expectsVisited(singleDevice)},
		{doubleID, expectsVisited(doubleDevice1, doubleDevice2)},
		{manyID, expectsVisited(manyDevice1, manyDevice2, manyDevice3, manyDevice4, manyDevice5)},
	}

	for _, record := range testData {
		t.Logf("%#v", record)
		registry := testRegistry(t, assert)
		actualVisited := visitedDevices{}

		assert.Equal(
			len(record.expectVisited),
			registry.visitID(record.expectedID, actualVisited.capture()),
		)

		assert.Equal(record.expectVisited, actualVisited)
	}
}

func TestRegistryVisitKey(t *testing.T) {
	assert := assert.New(t)
	testData := []struct {
		expectedKey   Key
		expectVisited visitedDevices
	}{
		{nosuchKey, expectsVisited()},
		{singleKey, expectsVisited(singleDevice)},
		{doubleKey1, expectsVisited(doubleDevice1)},
		{doubleKey2, expectsVisited(doubleDevice2)},
		{manyKey1, expectsVisited(manyDevice1)},
		{manyKey3, expectsVisited(manyDevice3)},
		{manyKey5, expectsVisited(manyDevice5)},
	}

	for _, record := range testData {
		t.Logf("%#v", record)
		registry := testRegistry(t, assert)
		actualVisited := visitedDevices{}

		assert.Equal(
			len(record.expectVisited),
			registry.visitKey(record.expectedKey, actualVisited.capture()),
		)

		assert.Equal(record.expectVisited, actualVisited)
	}
}

func TestRegistryVisitIf(t *testing.T) {
	assert := assert.New(t)
	testData := []struct {
		filter        func(ID) bool
		expectVisited visitedDevices
	}{
		{func(ID) bool { return false }, expectsVisited()},
		{func(id ID) bool { return id == singleID }, expectsVisited(singleDevice)},
		{func(id ID) bool { return id == doubleID }, expectsVisited(doubleDevice1, doubleDevice2)},
		{func(id ID) bool { return id == manyID }, expectsVisited(manyDevice1, manyDevice2, manyDevice3, manyDevice4, manyDevice5)},
	}

	for _, record := range testData {
		t.Logf("%#v", record)
		registry := testRegistry(t, assert)
		actualVisited := visitedDevices{}

		assert.Equal(
			len(record.expectVisited),
			registry.visitIf(record.filter, actualVisited.capture()),
		)

		assert.Equal(record.expectVisited, actualVisited)
	}
}

func TestRegistryVisitAll(t *testing.T) {
	assert := assert.New(t)
	registry := testRegistry(t, assert)

	expectVisited := expectsVisited(singleDevice, doubleDevice1, doubleDevice2, manyDevice1, manyDevice2, manyDevice3, manyDevice4, manyDevice5)

	actualVisited := visitedDevices{}
	assert.Equal(len(expectVisited), registry.visitAll(actualVisited.capture()))
	assert.Equal(expectVisited, actualVisited)
}

func TestRegistryAddDuplicateKey(t *testing.T) {
	assert := assert.New(t)
	registry := testRegistry(t, assert)
	duplicate := newDevice(singleID, singleKey, nil, 1)
	if addError := registry.add(duplicate); assert.NotNil(addError) {
		if duplicateKeyError, ok := addError.(DeviceError); assert.True(ok) {
			assert.Equal(invalidID, duplicateKeyError.ID())
			assert.Equal(singleKey, duplicateKeyError.Key())
		}
	}
}

func TestRegistryRemoveOne(t *testing.T) {
	assert := assert.New(t)
	testData := []struct {
		deviceToRemove *device
		expectRemove   bool
		expectVisitID  visitedDevices
		expectVisitAll visitedDevices
	}{
		{
			nosuchDevice,
			false,
			expectsVisited(),
			expectsVisited(singleDevice, doubleDevice1, doubleDevice2, manyDevice1, manyDevice2, manyDevice3, manyDevice4, manyDevice5),
		},
		{
			singleDevice,
			true,
			expectsVisited(),
			expectsVisited(doubleDevice1, doubleDevice2, manyDevice1, manyDevice2, manyDevice3, manyDevice4, manyDevice5),
		},
		{
			doubleDevice1,
			true,
			expectsVisited(doubleDevice2),
			expectsVisited(singleDevice, doubleDevice2, manyDevice1, manyDevice2, manyDevice3, manyDevice4, manyDevice5),
		},
		{
			manyDevice4,
			true,
			expectsVisited(manyDevice1, manyDevice2, manyDevice3, manyDevice5),
			expectsVisited(singleDevice, doubleDevice1, doubleDevice2, manyDevice1, manyDevice2, manyDevice3, manyDevice5),
		},
	}

	for _, record := range testData {
		t.Logf("%v", record)
		registry := testRegistry(t, assert)
		assert.Equal(record.expectRemove, registry.removeOne(record.deviceToRemove))

		actualVisitID := make(visitedDevices)
		registry.visitID(record.deviceToRemove.id, actualVisitID.capture())
		assert.Equal(record.expectVisitID, actualVisitID)

		actualVisitAll := make(visitedDevices)
		registry.visitAll(actualVisitAll.capture())
		assert.Equal(record.expectVisitAll, actualVisitAll)
	}
}

func TestRegistryRemoveAll(t *testing.T) {
	assert := assert.New(t)
	testData := []struct {
		idToRemove     ID
		expectRemoved  visitedDevices
		expectVisitAll visitedDevices
	}{
		{
			nosuchID,
			expectsVisited(),
			expectsVisited(singleDevice, doubleDevice1, doubleDevice2, manyDevice1, manyDevice2, manyDevice3, manyDevice4, manyDevice5),
		},
		{
			singleID,
			expectsVisited(singleDevice),
			expectsVisited(doubleDevice1, doubleDevice2, manyDevice1, manyDevice2, manyDevice3, manyDevice4, manyDevice5),
		},
		{
			doubleID,
			expectsVisited(doubleDevice1, doubleDevice2),
			expectsVisited(singleDevice, manyDevice1, manyDevice2, manyDevice3, manyDevice4, manyDevice5),
		},
		{
			manyID,
			expectsVisited(manyDevice1, manyDevice2, manyDevice3, manyDevice4, manyDevice5),
			expectsVisited(singleDevice, doubleDevice1, doubleDevice2),
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

		actualVisitAll := make(visitedDevices)
		registry.visitAll(actualVisitAll.capture())
		assert.Equal(record.expectVisitAll, actualVisitAll)
	}
}
