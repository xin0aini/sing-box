package script

type logWriter struct {
	loggerFunc func(args ...any)
}

func (w *logWriter) Write(p []byte) (n int, err error) {
	w.loggerFunc(string(p))
	return len(p), nil
}
