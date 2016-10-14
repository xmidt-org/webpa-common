package device

type registry map[ID][]*device

func (r registry) add(device *device) {
	key := device.id
	devices := r[key]
	r[key] = append(devices, device)
}

func (r registry) removeOne(device *device) {
	key := device.id
	devices := r[key]

	if len(devices) == 1 && devices[0] == device {
		delete(r, key)
		return
	}

	for index, candidate := range devices {
		if candidate == device {
			r[key] = append(devices[:index], devices[index+1:]...)
			return
		}
	}
}

func (r registry) removeAll(key ID) []*device {
	if devices, ok := r[key]; ok {
		delete(r, key)
		return devices
	}

	return nil
}
