package outbound

type torDebugWriter struct {
	t *Tor
}

func (t *torDebugWriter) Write(p []byte) (n int, err error) {
	t.t.logger.Debug(string(p))
	return len(p), nil
}
