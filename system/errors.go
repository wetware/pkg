package system

type Error struct {
	Module string
	Cause  error
}

func (err Error) Error() string {
	return err.Module + ": " + err.Cause.Error()
}

func (err Error) Unwrap() error {
	return err.Cause
}
