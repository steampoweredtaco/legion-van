/*
gui package contains the main vanity gui interface that provides convinence and channels to windows.
*/
package gui

import (
	"context"
	"fmt"
	"image"
	"sync"
	"sync/atomic"
	"time"

	"code.rocketnine.space/tslocum/cview"
	"github.com/gdamore/tcell/v2"
	"github.com/sirupsen/logrus"
)

type MonkeyPreview struct {
	Image image.Image
	Title string
}

type stats struct {
	total   uint64
	found   uint64
	started time.Time
}

type MainApp struct {
	app          *cview.Application
	speed        *cview.TextView
	totals       *cview.TextView
	runtimeStats stats
	ctx          context.Context
	mainCancel   context.CancelFunc
	preview      [4]*imageBox
	previewChan  chan MonkeyPreview
	log          *logrus.Logger
	previewWG    *sync.WaitGroup
	mainDoneChan chan struct{}
	totalDeltas  chan uint64
	once         sync.Once
}

func (a *MainApp) TotalDeltaChan() chan<- uint64 {
	return a.totalDeltas
}
func (a *MainApp) SetPreview(index int, img image.Image, title string) {
	a.preview[index].setImage(img)
	a.preview[index].SetTitle(title)
	a.preview[index].SetTitleAlign(cview.AlignCenter)
}

func (a *MainApp) UpdateTotal(additional uint64) {
	atomic.AddUint64(&a.runtimeStats.total, additional)
}

func (a *MainApp) UpdateSpeed() {
	total := atomic.LoadUint64(&a.runtimeStats.total)
	duration := time.Since(a.runtimeStats.started)
	if duration == 0 {
		a.speed.SetText("they've gone deplaid")
	}
	tps := float64(total) / float64(duration.Seconds())
	statText := fmt.Sprintf("%.2f per second", tps)
	a.speed.SetText(statText)
	a.app.QueueUpdateDraw(func() {}, a.speed)
}

func (a *MainApp) SetTerminalScreen(s tcell.Screen) {
	a.app.SetScreen(s)
}

// PNGPreviewChan is a channel png images can be piped to and they
// will go to the appropriate preivew panes.
func (a *MainApp) PNGPreviewChan() chan<- MonkeyPreview {
	if a.previewChan == nil {
		panic("gui isn't initialized")
	}
	return a.previewChan
}

func NewMainApp(ctx context.Context, mainCancel context.CancelFunc, title string, log *logrus.Logger) *MainApp {

	mainApp := new(MainApp)
	mainApp.previewChan = make(chan MonkeyPreview, 4)
	mainApp.mainDoneChan = make(chan struct{})
	mainApp.totalDeltas = make(chan uint64, 20)
	if log != nil {
		mainApp.log = log
	}

	mainApp.app = cview.NewApplication()

	flexBox := cview.NewFlex()
	flexBox.SetFullScreen(true)
	flexBox.SetTitle(title)
	flexBox.SetBorder(true)
	flexBox.SetDirection(cview.FlexRow)
	body := cview.NewFlex()
	for i := 0; i < 4; i++ {
		imgBox := NewImageBox(nil, 30, 10)
		imgBox.SetBorder(true)
		body.AddItem(imgBox, 0, 1, false)
		mainApp.preview[i] = imgBox
	}
	flexBox.AddItem(body, 0, 3, false)

	footer := cview.NewFlex()
	speed := cview.NewTextView()
	speed.SetTextAlign(cview.AlignRight)
	speed.SetText("engaged")
	mainApp.speed = speed
	total := cview.NewTextView()
	total.SetTextAlign(cview.AlignLeft)
	total.SetText("readying the hordes")
	footer.AddItem(total, 0, 1, false)
	footer.AddItem(speed, 0, 1, false)
	flexBox.AddItem(footer, 1, 0, false)
	mainApp.app.SetRoot(flexBox, true)
	captureHandler := func(event *tcell.EventKey) *tcell.EventKey {
		return mainApp.HandleEvent(event)
	}

	mainApp.app.SetInputCapture(captureHandler)

	mainApp.ctx, mainApp.mainCancel = ctx, mainCancel
	mainApp.runtimeStats.started = time.Now()
	return mainApp
}

// ForceDebugResize is required when adding a previous screen because widht and height will not be setup correct, this seems like a bug.
func (a *MainApp) ForceDebugResize() {
	w, h := a.app.GetScreen().Size()
	event := tcell.NewEventResize(w, h)
	a.app.QueueEvent(event)
}

func (mainApp *MainApp) Done() <-chan struct{} {
	return mainApp.ctx.Done()
}

// MainHasShtudown notifies the gui that the main app and use of any sender channels from
// the mainapp are not in use any longer.  This is required before the Quit methods will work
// because mainApp is responsible for shutting down the gui sender chans, such as preview, that
// the main app controls the lifetime of.
func (mainApp *MainApp) MainHasShutdown() {
	logrus.Info("Notifed main has shutdown")
	mChan := mainApp.mainDoneChan
	mChan <- struct{}{}
	close(mChan)
}

func (mainApp *MainApp) ForceQuit() {
	mainApp.app.QueueUpdate(func() { mainApp.app.Stop() })
}

func (m *MainApp) Quit() {
	// notify all users of the gui before shutting down the gui
	logrus.Info("Canceling main")
	m.mainCancel()
	// Cannot cleanup send channels from main until verification that main is done using them
main:
	for {
		deadline := time.NewTimer(time.Second * 10)
		mainDoneChan := m.mainDoneChan
		select {
		case <-mainDoneChan:
			logrus.Info("Shutting down main gui after main app has reported done")
			break main
		case <-deadline.C:
			logrus.Info("tick")
		}

	}
	logrus.Info("Closing channel")
	close(m.previewChan)
	logrus.Info("Sending gui queue to stop qui")
	m.app.QueueUpdate(func() { m.app.Stop() })

}

func (m *MainApp) processPreviews() {
	defer m.previewWG.Done()
	nextPreviewPane := uint(0)

	for {
		select {
		case <-m.ctx.Done():
			logrus.Debug("shutting down preview listener")
			return
		case monkeyPreview := <-m.previewChan:
			imageBox := m.preview[nextPreviewPane%4]
			imageBox.setImage(monkeyPreview.Image)
			imageBox.SetTitle(monkeyPreview.Title)
			nextPreviewPane++
			// We want to wait for preview completion before pushing out
			// another
			wg := new(sync.WaitGroup)
			wg.Add(1)
			logrus.Infof("Previewing %s", monkeyPreview.Title)
			m.app.QueueUpdateDraw(func() { wg.Done() }, imageBox)
			wg.Wait()
		}
	}
}

func (m *MainApp) listenForPreviews() {
	m.previewWG = new(sync.WaitGroup)
	// TODO make the number of preview panes configurable.
	m.previewWG.Add(1)
	go m.processPreviews()
}

func (m *MainApp) cleanupGui() {
	m.previewWG.Wait()
}

func (m *MainApp) Run() {
	m.listenForPreviews()
	go func(doneChan <-chan struct{}) {
		ticker := time.NewTicker(time.Second)
		for {
			select {
			case <-ticker.C:
				m.UpdateSpeed()
			case delta := <-m.totalDeltas:
				m.UpdateTotal(delta)
			case <-doneChan:
				return
			}
		}
	}(m.mainDoneChan)

	m.app.Run()
	m.cleanupGui()
}

func (m *MainApp) DebugPressEsc() {
	m.once.Do(func() { m.Quit() })
}
func (mainApp *MainApp) HandleEvent(event *tcell.EventKey) *tcell.EventKey {
	if event.Key() == tcell.KeyEscape {
		go mainApp.once.Do(func() { mainApp.Quit() })
		return nil
	}
	return event
}
