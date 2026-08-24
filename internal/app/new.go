package app

import "os/exec"

func NewManager() *Manager {
	return &Manager{
		procs:  map[string]*exec.Cmd{},
		errors: map[string]string{},
	}
}
