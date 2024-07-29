package main

import (
	_ "embed"
	"strconv"
	"sync/atomic"
	"time"

	"context"

	"github.com/getlantern/systray"
	"github.com/go-vgo/robotgo"
	"github.com/ncruces/zenity"
)

//go:embed logo.ico
var logo []byte

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	robotgo.Scale = true

	i := &atomic.Int64{}
	i.Store(5)
	m := mouse{
		interval: i,
		logo:     logo,
		cancel:   cancel,
		ctx:      ctx,
	}
	go m.do(ctx)
	systray.Run(m.onReady, m.onExit)
}

type mouse struct {
	interval *atomic.Int64
	logo     []byte
	cancel   func()
	ctx      context.Context
}

func (m *mouse) onReady() {
	systray.SetIcon(m.logo)
	systray.SetTitle("鼠标抖动")
	systray.SetTooltip("鼠标抖动")

	timeCh := systray.AddMenuItem("修改间隔时间", "")

	quitCh := systray.AddMenuItem("退出", "")
	go func() {
		select {
		case <-m.ctx.Done():
			return
		case <-quitCh.ClickedCh:
			systray.Quit()
		}
	}()

	for {
		select {
		case <-timeCh.ClickedCh:
			i, err := zenity.Entry("", zenity.Title("间隔时间（秒）"), zenity.EntryText(strconv.FormatInt(m.interval.Load(), 10)))
			if err == nil {
				inter, err := strconv.ParseInt(i, 10, 64)
				if err != nil {
					zenity.Warning(err.Error())
					continue
				}
				m.interval.Store(inter)
			}
		case <-m.ctx.Done():
			return
		}
	}

}

func (m *mouse) onExit() {
	systray.Quit()
	m.cancel()
}

func (m *mouse) do(ctx context.Context) {
	has := false
	for {
		if has {
			robotgo.MoveRelative(1, 1)
			has = false
		} else {
			robotgo.MoveRelative(-1, -1)
			has = true
		}
		time.Sleep(time.Duration(m.interval.Load()) * time.Second)

		select {
		case <-ctx.Done():
			return
		default:
		}
	}
}
