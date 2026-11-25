package settings

// migrate brings a configuration written by an older build forward. It only
// replaces values that still hold the old defaults, so a deliberate schedule
// is kept.
func migrate(s *Settings) {
	if s.Version >= CurrentVersion {
		return
	}
	oldSchedule := s.Schedule.Mini.DurationMs == 20_000 &&
		s.Schedule.Mini.IntervalMs == 600_000 &&
		s.Schedule.Long.DurationMs == 300_000 &&
		s.Schedule.Long.EveryNBreaks == 2
	if oldSchedule {
		def := Default().Schedule
		s.Schedule.Mini.DurationMs = def.Mini.DurationMs
		s.Schedule.Mini.IntervalMs = def.Mini.IntervalMs
		s.Schedule.Long.DurationMs = def.Long.DurationMs
		s.Schedule.Long.EveryNBreaks = def.Long.EveryNBreaks
		s.Schedule.Long.IntervalMs = def.Long.IntervalMs
	}
	// A break held for typing was easy to mistake for a stuck state; it is
	// opt-in again.
	s.Presence.WaitWhileTyping = false
	s.Version = CurrentVersion
}
