package device

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

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

func TestRegistryVisitKey(t *testing.T) {
	assert := assert.New(t)
	testData := []struct {
		expectedKey   Key
		expectVisited deviceSet
	}{
		{nosuchKey, expectsDevices()},
		{singleKey, expectsDevices(singleDevice)},
		{doubleKey1, expectsDevices(doubleDevice1)},
		{doubleKey2, expectsDevices(doubleDevice2)},
		{manyKey1, expectsDevices(manyDevice1)},
		{manyKey3, expectsDevices(manyDevice3)},
		{manyKey5, expectsDevices(manyDevice5)},
	}

	for _, record := range testData {
		t.Logf("%#v", record)
		registry := testRegistry(t, assert)
		actualVisited := deviceSet{}

		assert.Equal(
			len(record.expectVisited),
			registry.visitKey(record.expectedKey, actualVisited.registryCapture()),
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
		expectVisitID  deviceSet
		expectVisitAll deviceSet
	}{
		{
			nosuchDevice,
			false,
			expectsDevices(),
			expectsDevices(singleDevice, doubleDevice1, doubleDevice2, manyDevice1, manyDevice2, manyDevice3, manyDevice4, manyDevice5),
		},
		{
			singleDevice,
			true,
			expectsDevices(),
			expectsDevices(doubleDevice1, doubleDevice2, manyDevice1, manyDevice2, manyDevice3, manyDevice4, manyDevice5),
		},
		{
			doubleDevice1,
			true,
			expectsDevices(doubleDevice2),
			expectsDevices(singleDevice, doubleDevice2, manyDevice1, manyDevice2, manyDevice3, manyDevice4, manyDevice5),
		},
		{
			manyDevice4,
			true,
			expectsDevices(manyDevice1, manyDevice2, manyDevice3, manyDevice5),
			expectsDevices(singleDevice, doubleDevice1, doubleDevice2, manyDevice1, manyDevice2, manyDevice3, manyDevice5),
		},
	}

	for _, record := range testData {
		t.Logf("%v", record)
		registry := testRegistry(t, assert)
		assert.Equal(record.expectRemove, registry.removeOne(record.deviceToRemove))

		actualVisitID := make(deviceSet)
		registry.visitID(record.deviceToRemove.id, actualVisitID.registryCapture())
		assert.Equal(record.expectVisitID, actualVisitID)

		actualVisitAll := make(deviceSet)
		registry.visitAll(actualVisitAll.registryCapture())
		assert.Equal(record.expectVisitAll, actualVisitAll)
	}
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
