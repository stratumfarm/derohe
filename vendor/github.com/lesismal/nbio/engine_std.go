// Copyright 2020 lesismal. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

//go:build windows
// +build windows

package nbio

import (
	"runtime"
	"strings"
	"time"

	"github.com/lesismal/nbio/logging"
)

// Start init and start pollers
func (g *Engine) Start() error {
	var err error

	g.listeners = make([]*poller, len(g.addrs))
	for i := range g.addrs {
		g.listeners[i], err = newPoller(g, true, int(i))
		if err != nil {
			for j := 0; j < i; j++ {
				g.listeners[j].stop()
			}
			return err
		}
	}

	for i := 0; i < g.pollerNum; i++ {
		g.pollers[i], err = newPoller(g, false, int(i))
		if err != nil {
			for j := 0; j < len(g.addrs); j++ {
				g.listeners[j].stop()
			}

			for j := 0; j < int(i); j++ {
				g.pollers[j].stop()
			}
			return err
		}
	}

	for i := 0; i < g.pollerNum; i++ {
		g.Add(1)
		go g.pollers[i].start()
	}
	for _, l := range g.listeners {
		g.Add(1)
		go l.start()
	}

	g.Add(1)
	go g.timerLoop()

	if len(g.addrs) == 0 {
		logging.Info("NBIO[%v] start", g.Name)
	} else {
		logging.Info("NBIO[%v] start listen on: [\"%v\"]", g.Name, strings.Join(g.addrs, `", "`))
	}
	return nil
}

// NewEngine is a factory impl
func NewEngine(conf Config) *Engine {
	cpuNum := runtime.NumCPU()
	if conf.Name == "" {
		conf.Name = "NB"
	}
	if conf.NPoller <= 0 {
		conf.NPoller = cpuNum
	}
	if conf.ReadBufferSize <= 0 {
		conf.ReadBufferSize = DefaultReadBufferSize
	}

	g := &Engine{
		Name:               conf.Name,
		network:            conf.Network,
		addrs:              conf.Addrs,
		pollerNum:          conf.NPoller,
		readBufferSize:     conf.ReadBufferSize,
		maxWriteBufferSize: conf.MaxWriteBufferSize,
		lockListener:       conf.LockListener,
		lockPoller:         conf.LockPoller,
		listeners:          make([]*poller, len(conf.Addrs)),
		pollers:            make([]*poller, conf.NPoller),
		connsStd:           map[*Conn]struct{}{},
		callings:           []func(){},
		chCalling:          make(chan struct{}, 1),
		trigger:            time.NewTimer(timeForever),
		chTimer:            make(chan struct{}),
	}

	g.initHandlers()

	g.OnReadBufferAlloc(func(c *Conn) []byte {
		if c.ReadBuffer == nil {
			c.ReadBuffer = make([]byte, int(g.readBufferSize))
		}
		return c.ReadBuffer
	})

	return g
}
