package app

import (
	"time"

	"github.com/vdatanet/atrium-go/internal/ports"
	"github.com/vdatanet/atrium-go/internal/units"
)

// systemClock is the wall clock, as ports.Clock.
//
// It lives in the entry layer rather than in internal/ports because ports holds
// interfaces and nothing else, and because the concrete clock belongs where it
// is wired: architecture 2's argument for making the clock a port at all is
// that "a clock the tests replace is what keeps a golden body stable between
// two runs", and a package that both declared the port and shipped the real
// implementation would make the real one the path of least resistance.
//
// It rounds to a whole tick, because units.At does. That is not a detail: every
// date this server sends is .NET ticks on the wire (behaviours 1.3), so an
// instant this process could hold and not spell would be a value that changed
// on its way out.
type systemClock struct{}

func (systemClock) Now() units.Time { return units.At(time.Now()) }

// SystemClock is this process's wall clock.
//
// It is a function rather than an exported variable so that nothing can replace
// the real clock at a distance: a test that wants a clock it controls passes
// its own ports.Clock to whatever it is testing.
func SystemClock() ports.Clock { return systemClock{} }
