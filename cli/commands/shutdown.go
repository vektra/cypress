package commands

type ShutdownHandler interface {
	OnShutdown(f func())
}

var shutdown ShutdownHandler

func SetShutdownHandler(h ShutdownHandler) {
	shutdown = h
}

func OnShutdown(f func()) {
	if shutdown != nil {
		shutdown.OnShutdown(f)
	}
}
