package main

import (
	_ "embed"
	"fmt"
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

	mCtx, mCancel := context.WithCancel(ctx)
	m := mouse{
		interval: i,
		logo:     logo,
		cancel:   cancel,
		ctx:      ctx,
		mCancel:  mCancel,
	}
	go m.do(mCtx)
	systray.Run(m.onReady, m.onExit)
}

type mouse struct {
	interval *atomic.Int64
	logo     []byte
	cancel   func()
	mCancel  func()
	ctx      context.Context
}

func (m *mouse) onReady() {
	systray.SetIcon(m.logo)
	systray.SetTooltip("鼠标抖动\n间隔时间: 5 秒")

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
				m.mCancel()
				ctx, cancel := context.WithCancel(m.ctx)
				m.mCancel = cancel
				go m.do(ctx)
				systray.SetTooltip(fmt.Sprintf("鼠标抖动\n间隔时间: %v 秒", inter))
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
	tick := time.NewTicker(time.Duration(m.interval.Load()) * time.Second)
	defer tick.Stop()

	move := int(robotgo.ScaleF() * 1)

	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			if has {

				robotgo.MoveRelative(move, move)
				has = false
			} else {
				robotgo.MoveRelative(-move, -move)
				has = true
			}
		}
	}
}
