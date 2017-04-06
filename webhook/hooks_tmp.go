package webhook

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"time"
)

const (
	HOOKS_UPDATE_INTERVAL time.Duration = time.Second * 300
	HOOKS_JSON_FILE                     = "hooks-list.json"
)

func LoadHooksList() []W {
	// Load hoooks-list.json and populate in []W
	file, err := ioutil.ReadFile("hooks-list.json")
	if nil != err {
		fmt.Printf("Failed to read file: %v\n", err)
		return nil
	}
	var hooks []W
	json_err := json.Unmarshal(file, &hooks)
	if json_err != nil {
		fmt.Println("error:", json_err)
	}
	fmt.Println("%+v", hooks)
	return hooks
}

func extendHooksDuration(m *monitor) []W {
	var update []W
	len := m.list.Len()
	update = make([]W, len)
	i := 0
	for i < len {
		update = append(update, *(m.list.Get(i)))
		update[i].Until = time.Now().Add(HOOKS_UPDATE_INTERVAL).Unix()
	}
	return update
}

func updateHooksTicker(m *monitor) {
	var updatedHooks []W
	ticker := time.NewTicker(HOOKS_UPDATE_INTERVAL)
	for t := range ticker.C {
		fmt.Println("Tick at", t)
		updatedHooks = extendHooksDuration(m)
	}
	m.list.Update(updatedHooks)
}
