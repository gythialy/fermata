package presence

import (
	"github.com/egoist/mygo"
)

// SystemSource reads the system: MyGo's power and idle APIs, the platform's
// Focus probe, and the process list.
func SystemSource() Source {
	return Source{
		IdleTime: mygo.Power.IdleTime,
		ProbeDND: ProbeDND,
		Running:  RunningProcesses,
		Power:    true,
		Dispatch: mygo.RunOnMain,
	}
}
