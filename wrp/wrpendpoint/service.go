package wrpendpoint

// Service represents a component which processes WRP transactions.
type Service interface {
	// ServeWRP processes a WRP request
	ServeWRP(Request) (Response, error)
}

// ServiceFunc is a function type that implements Service
type ServiceFunc func(Request) (Response, error)

func (sf ServiceFunc) ServeWRP(r Request) (Response, error) {
	return sf(r)
}
