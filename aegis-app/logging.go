package main

import (
	"log"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) logInfof(format string, args ...interface{}) {
	if a.ctx != nil {
		runtime.LogInfof(a.ctx, format, args...)
	} else {
		log.Printf("[INFO] "+format, args...)
	}
}

func (a *App) logWarningf(format string, args ...interface{}) {
	if a.ctx != nil {
		runtime.LogWarningf(a.ctx, format, args...)
	} else {
		log.Printf("[WARN] "+format, args...)
	}
}

func (a *App) logErrorf(format string, args ...interface{}) {
	if a.ctx != nil {
		runtime.LogErrorf(a.ctx, format, args...)
	} else {
		log.Printf("[ERROR] "+format, args...)
	}
}

func (a *App) emitEvent(name string, data ...interface{}) {
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, name, data...)
	}
}
