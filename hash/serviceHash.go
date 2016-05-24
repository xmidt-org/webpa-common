package hash

import (
	"errors"
	"sync/atomic"
	"unsafe"
)

var (
	ServiceHashHolderUninitialized = errors.New("ServiceHashHolder is not initialized")
)

// ServiceHash represents a component which can return URLs as strings based
// on arbitrary keys.
type ServiceHash interface {
	// Get returns the service URL associated with the given key.
	Get([]byte) (string, error)
}

// ServiceHashHolder represents an atomic pointer to a ServiceHash.  This pointer
// can be used without locking.  This type implements ServiceHash.  The current
// reference is used via the ServiceHash interface.
type ServiceHashHolder struct {
	current unsafe.Pointer
}

var _ ServiceHash = (*ServiceHashHolder)(nil)

// reference returns the current, unsafe reference
func (holder *ServiceHashHolder) reference() *ServiceHash {
	return (*ServiceHash)(atomic.LoadPointer(&holder.current))
}

func (holder *ServiceHashHolder) Get(key []byte) (string, error) {
	reference := holder.reference()
	if reference == nil {
		return "", ServiceHashHolderUninitialized
	}

	return (*reference).Get(key)
}

// Update atomically updates the current ServiceHash instance.  Subsequent calls to Get()
// will use the newHash instance.
func (holder *ServiceHashHolder) Update(newHash ServiceHash) {
	atomic.StorePointer(&holder.current, unsafe.Pointer(&newHash))
}

// Connected checks whether this holder has any hash entries.
// This implementations handler.Connection, which allows this holder
// to participate in request gating.
func (holder *ServiceHashHolder) Connected() bool {
	return holder.reference() != nil
}
