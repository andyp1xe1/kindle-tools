package jsrepl

import "time"

const (
	Name = "jsrepl"
	Port = 6767

	longPollWait = 25 * time.Second
	maxQueue     = 100
	maxResults   = 200
)
