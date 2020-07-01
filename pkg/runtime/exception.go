package runtime

// Exception is a boxed error.
type Exception struct {
	error
	fs map[string]interface{} // immutable
}

// Loggable representation of the error
func (exc Exception) Loggable() map[string]interface{} {
	m := make(map[string]interface{}, len(exc.fs)+1)
	m["error"] = exc.error
	for k, v := range exc.fs {
		m[k] = v
	}
	return m
}
