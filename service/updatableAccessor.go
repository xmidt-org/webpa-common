package service

var (
//ErrAccessorUninitialized = errors.New("Accessor has not been initialized")
)

/*
// UpdatableAccessor represents an Accessor whose state can be updated.
// Another Accessor is delegated to for Get calls, and this Accessor can
// be changed via Update.
type UpdatableAccessor struct {
	lock    sync.RWMutex
	current Accessor
}

// Get uses the current Accessor delegate to hash the key.  This method
// returns ErrAccessorUninitialized if there is no current Accessor (yet).
//
// It is safe to invoke this method concurrently with itself or Update.
func (u *UpdatableAccessor) Get(key []byte) (instance string, err error) {
	u.lock.RLock()
	if u.current == nil {
		err = ErrAccessorUninitialized
	} else {
		instance, err = u.current.Get(key)
	}

	u.lock.RUnlock()
	return
}

// Update changes the current Accessor delegate.  It is legal to call Update(nil),
// in which case Get will return ErrAccessorUninitialized.
//
// It is safe to invoke this method concurrently with itself or Get.
func (u *UpdatableAccessor) Update(a Accessor) {
	u.lock.Lock()
	u.current = a
	u.lock.Unlock()
}
*/
