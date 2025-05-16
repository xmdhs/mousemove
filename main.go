package main

import (
	_ "embed"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"

	"context"

	"github.com/getlantern/systray"
	"github.com/go-vgo/robotgo"
	"github.com/ncruces/zenity"
	"golang.org/x/sys/windows"
)

var (
	user32                   = windows.NewLazySystemDLL("user32.dll")
	procEnumWindows          = user32.NewProc("EnumWindows")
	procGetWindowTextW       = user32.NewProc("GetWindowTextW")
	procGetWindowTextLengthW = user32.NewProc("GetWindowTextLengthW")
	procIsWindowVisible      = user32.NewProc("IsWindowVisible")
	procShowWindow           = user32.NewProc("ShowWindow")
)

const (
	SW_MINIMIZE = 6
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
	go run(ctx)
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

// 枚举窗口的回调函数
type enumWindowsProc func(hwnd syscall.Handle, lParam uintptr) bool

func enumWindows(callback enumWindowsProc, lParam uintptr) error {
	cb := syscall.NewCallback(func(hwnd syscall.Handle, lParam uintptr) uintptr {
		if callback(hwnd, lParam) {
			return 1 // continue
		}
		return 0 // stop
	})
	ret, _, err := procEnumWindows.Call(cb, lParam)
	if ret == 0 {
		return err
	}
	return nil
}

// 获取窗口标题
func getWindowText(hwnd syscall.Handle) string {
	length, _, _ := procGetWindowTextLengthW.Call(uintptr(hwnd))
	if length == 0 {
		return ""
	}
	buf := make([]uint16, length+1)
	procGetWindowTextW.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&buf[0])), length+1)
	return syscall.UTF16ToString(buf)
}

// 判断窗口是否可见
func isWindowVisible(hwnd syscall.Handle) bool {
	ret, _, _ := procIsWindowVisible.Call(uintptr(hwnd))
	return ret != 0
}

// 最小化窗口
func minimizeWindow(hwnd syscall.Handle) {
	procShowWindow.Call(uintptr(hwnd), SW_MINIMIZE)
}

func run(ctx context.Context) {
	c := time.Tick(1 * time.Minute)
	for {
		hidesList := []syscall.Handle{}
		needHide := true

		err := enumWindows(func(hwnd syscall.Handle, lParam uintptr) bool {
			if !isWindowVisible(hwnd) {
				return true
			}
			title := getWindowText(hwnd)
			if strings.Contains(title, "企业微信") {
				hidesList = append(hidesList, hwnd)
			}
			if strings.Contains(title, "RustDesk") {
				needHide = false
			}
			return true
		}, 0)
		if err != nil {
			log.Println("发生错误：", err)
		}
		if needHide {
			for _, v := range hidesList {
				minimizeWindow(v)
			}
		}

		select {
		case <-c:
		case <-ctx.Done():
			return
		}
	}
}
