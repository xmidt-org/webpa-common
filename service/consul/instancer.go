package consul

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"sync"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/go-kit/kit/sd"
	"github.com/go-kit/kit/util/conn"
	"github.com/hashicorp/consul/api"
	"github.com/xmidt-org/webpa-common/v2/logging"
)

var (
	errStopped = errors.New("Instancer stopped")
)

type InstancerOptions struct {
	Client       Client
	Logger       log.Logger
	Service      string
	Tags         []string
	PassingOnly  bool
	QueryOptions api.QueryOptions
}

func NewInstancer(o InstancerOptions) sd.Instancer {
	if o.Logger == nil {
		o.Logger = logging.DefaultLogger()
	}

	i := &instancer{
		client:       o.Client,
		logger:       log.With(o.Logger, "service", o.Service, "tags", fmt.Sprint(o.Tags), "passingOnly", o.PassingOnly, "datacenter", o.QueryOptions.Datacenter),
		service:      o.Service,
		passingOnly:  o.PassingOnly,
		queryOptions: o.QueryOptions,
		stop:         make(chan struct{}),
		registry:     make(map[chan<- sd.Event]bool),
	}

	if len(o.Tags) > 0 {
		i.tag = o.Tags[0]
		for ix := 1; ix < len(o.Tags); ix++ {
			i.filterTags = append(i.filterTags, o.Tags[ix])
		}
	}

	// grab the initial set of instances
	instances, index, err := i.getInstances(0, nil)
	if err == nil {
		i.logger.Log(level.Key(), level.InfoValue(), "instances", len(instances))
	} else {
		i.logger.Log(level.Key(), level.ErrorValue(), logging.ErrorKey(), err)
	}

	i.update(sd.Event{Instances: instances, Err: err})
	go i.loop(index)

	return i
}

type instancer struct {
	client  Client
	logger  log.Logger
	service string

	tag        string
	filterTags []string

	passingOnly  bool
	queryOptions api.QueryOptions

	stop chan struct{}

	registerLock sync.Mutex
	state        sd.Event
	registry     map[chan<- sd.Event]bool
}

func (i *instancer) update(e sd.Event) {
	sort.Strings(e.Instances)
	defer i.registerLock.Unlock()
	i.registerLock.Lock()

	if reflect.DeepEqual(i.state, e) {
		return
	}

	i.state = e
	for c := range i.registry {
		c <- i.state
	}
}

func (i *instancer) loop(lastIndex uint64) {
	var (
		instances []string
		err       error
		d         time.Duration = 10 * time.Millisecond
	)

	for {
		instances, lastIndex, err = i.getInstances(lastIndex, i.stop)
		switch {
		case err == errStopped:
			return

		case err != nil:
			i.logger.Log(logging.ErrorKey(), err)
			time.Sleep(d)
			d = conn.Exponential(d)
			i.update(sd.Event{Err: err})

		default:
			i.update(sd.Event{Instances: instances})
			d = 10 * time.Millisecond
		}
	}
}

// getInstances is implemented similarly to go-kits sd/consul version, albeit with support for
// arbitrary query options
func (i *instancer) getInstances(lastIndex uint64, stop <-chan struct{}) ([]string, uint64, error) {
	type response struct {
		instances []string
		index     uint64
		err       error
	}

	result := make(chan response, 1)

	go func() {
		var queryOptions api.QueryOptions = i.queryOptions
		queryOptions.WaitIndex = lastIndex
		entries, meta, err := i.client.Service(i.service, i.tag, i.passingOnly, &queryOptions)
		if err != nil {
			result <- response{err: err}
			return
		}

		if len(i.filterTags) > 0 {
			entries = filterEntries(entries, i.filterTags)
		}

		// see: https://www.consul.io/api-docs/features/blocking#implementation-details
		if meta == nil || meta.LastIndex < lastIndex {
			lastIndex = 0
		} else {
			lastIndex = meta.LastIndex
		}

		result <- response{
			instances: makeInstances(entries),
			index:     lastIndex,
		}
	}()

	select {
	case r := <-result:
		return r.instances, r.index, r.err
	case <-stop:
		return nil, 0, errStopped
	}
}

func filterEntry(candidate *api.ServiceEntry, requiredTags []string) bool {
	serviceTags := make(map[string]bool, len(candidate.Service.Tags))
	for _, tag := range candidate.Service.Tags {
		serviceTags[tag] = true
	}

	for _, requiredTag := range requiredTags {
		if !serviceTags[requiredTag] {
			return false
		}
	}

	return true
}

// filterEntries is similar to go-kit's version: since consul does not support multiple tags
// in blocking queries, we have to filter manually for multiple tags.
func filterEntries(entries []*api.ServiceEntry, requiredTags []string) []*api.ServiceEntry {
	var filtered []*api.ServiceEntry
	for _, entry := range entries {
		if filterEntry(entry, requiredTags) {
			filtered = append(filtered, entry)
		}
	}

	return filtered
}

// makeInstances is identical to go-kit's version
func makeInstances(entries []*api.ServiceEntry) []string {
	instances := make([]string, len(entries))
	for i, entry := range entries {
		address := entry.Node.Address
		if len(entry.Service.Address) > 0 {
			address = entry.Service.Address
		}

		instances[i] = fmt.Sprintf("%s:%d", address, entry.Service.Port)
	}

	return instances
}

func (i *instancer) Register(ch chan<- sd.Event) {
	defer i.registerLock.Unlock()
	i.registerLock.Lock()
	i.registry[ch] = true

	// push the current state to the new channel
	ch <- i.state
}

func (i *instancer) Deregister(ch chan<- sd.Event) {
	defer i.registerLock.Unlock()
	i.registerLock.Lock()
	delete(i.registry, ch)
}

func (i *instancer) Stop() {
	// this isn't idempotent, but mimics go-kit's behavior
	close(i.stop)
}
