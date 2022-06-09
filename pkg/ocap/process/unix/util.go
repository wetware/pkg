package unix

// writerFunc is a function type that implements io.Writer.
type writerFunc func([]byte) (int, error)

// Write calls the function with b as its argument.
func (write writerFunc) Write(b []byte) (int, error) {
	return write(b)
}
