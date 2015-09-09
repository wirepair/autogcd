package autogcd

import (
	"github.com/wirepair/gcd/gcdprotogen/types"
)

type PageLoadEventFired struct {
	timestamp int
}

type ConsoleEventHeader struct {
	Method string              `json:"method"`
	Params *ConsoleEventParams `json:"params"`
}

type ConsoleEventParams struct {
	Message *types.ChromeConsoleConsoleMessage `json:"message"`
}
