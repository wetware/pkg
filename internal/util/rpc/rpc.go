package rpcutil

type ErrReporterFunc func(error)

func (f ErrReporterFunc) ReportError(err error) { f(err) }
