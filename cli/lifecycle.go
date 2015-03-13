package cli

import (
	"os"
	"os/signal"
	"sync"
	"syscall"
)

type LifecycleData struct {
	lock sync.Mutex

	ranShutdown bool
	onShutdown  []func()
}

var Lifecycle = &LifecycleData{}

func (l *LifecycleData) OnShutdown(handle func()) {
	l.lock.Lock()
	defer l.lock.Unlock()

	l.onShutdown = append(l.onShutdown, handle)
}

func (l *LifecycleData) Shutdown(code int) {
	l.RunCleanup()
	os.Exit(code)
}

func (l *LifecycleData) RunCleanup() {
	l.lock.Lock()
	defer l.lock.Unlock()

	if l.ranShutdown {
		return
	}

	l.ranShutdown = true

	for _, h := range l.onShutdown {
		h()
	}
}

func (l *LifecycleData) Start() {
	c := make(chan os.Signal)

	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)

	go l.watchTheThrone(c)
}

func (l *LifecycleData) watchTheThrone(c chan os.Signal) {
	<-c

	l.Shutdown(0)
}
