package autogcd

import (
	"github.com/wirepair/gcd/gcdapi"
)

type PageLoadEventFired struct {
	timestamp int
}

type ConsoleEventHeader struct {
	Method string              `json:"method"`
	Params *ConsoleEventParams `json:"params"`
}

type ConsoleEventParams struct {
	Message *gcdapi.ConsoleConsoleMessage `json:"message"`
}

type DefaultEventHeader struct {
	Method string      `json:"method"`
	Params interface{} `json:"params"`
}
