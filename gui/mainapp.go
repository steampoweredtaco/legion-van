/*
gui package contains the main vanity gui interface that provides convinence and channels to windows.
*/
package gui

import (
	"context"
	"fmt"
	"image"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"code.rocketnine.space/tslocum/cview"
	"github.com/gdamore/tcell/v2"
	"github.com/sirupsen/logrus"
	"github.com/steampoweredtaco/legion-van/engine"
)

type MonkeyPreview struct {
	Image image.Image
	Title string
}

type MainApp struct {
	app          *cview.Application
	speed        *cview.TextView
	total        *cview.TextView
	logview      *cview.TextView
	logChan      chan *logrus.Entry
	runtimeStats engine.Stats
	ctx          context.Context
	mainCancel   context.CancelFunc
	preview      [4]*imageBox
	previewChan  chan MonkeyPreview
	log          *logrus.Logger
	previewWG    *sync.WaitGroup
	statDeltas   chan engine.Stats
	once         sync.Once
	endTime      time.Time
}

func (a *MainApp) GetTotalStat() uint64 {
	return atomic.LoadUint64(&a.runtimeStats.Total)
}

func (a *MainApp) UpdateStats(stats engine.Stats) {
	a.UpdateTotalStat(stats.Total)
	a.UpdateFoundStat(stats.Found)
	a.UpdateTotalRequestsStat(stats.TotalRequests)
}

func (a *MainApp) TotalDeltaChan() chan<- engine.Stats {
	return a.statDeltas
}
func (a *MainApp) SetPreview(index int, img image.Image, title string) {
	a.preview[index].setImage(img)
	a.preview[index].SetTitle(title)
	a.preview[index].SetTitleAlign(cview.AlignCenter)
}

func (a *MainApp) UpdateTotalStat(additional uint64) {
	if additional == 0 {
		return
	}
	atomic.AddUint64(&a.runtimeStats.Total, additional)
}
func (a *MainApp) UpdateFoundStat(additional uint64) {
	if additional == 0 {
		return
	}
	atomic.AddUint64(&a.runtimeStats.Found, additional)
}
func (a *MainApp) UpdateTotalRequestsStat(additional uint64) {
	if additional == 0 {
		return
	}
	atomic.AddUint64(&a.runtimeStats.TotalRequests, additional)
}

func (a *MainApp) UpdateSpeed() {
	total := atomic.LoadUint64(&a.runtimeStats.Total)
	duration := time.Since(a.runtimeStats.Started)
	if duration == 0 {
		a.speed.SetText("they've gone deplaid")
	}
	tps := float64(total) / float64(duration.Seconds())
	statText := fmt.Sprintf("%.2f per second", tps)
	a.speed.SetText(statText)
	a.app.QueueUpdateDraw(func() {}, a.speed)
}

func (a *MainApp) UpdateTotal() {
	totalFound := atomic.LoadUint64(&a.runtimeStats.Found)
	total := atomic.LoadUint64(&a.runtimeStats.Total)
	totalRequests := atomic.LoadUint64(&a.runtimeStats.TotalRequests)

	until := time.Until(a.endTime)
	if until < time.Second*0 {
		until = time.Second * 0
	}
	statText := fmt.Sprintf("time left %s. raid parties: %d. raided: %d. looted: %d.",
		until.Round(time.Second), totalRequests, total, totalFound)
	a.total.SetText(statText)
	a.app.QueueUpdateDraw(func() {}, a.total)
}

func (a *MainApp) SetTerminalScreen(s tcell.Screen) {
	a.app.SetScreen(s)
}

// PNGPreviewChan is a channel png images can be piped to and they
// will go to the appropriate preivew panes.
func (a *MainApp) PNGPreviewChan() (monkeyPreviewChan chan<- MonkeyPreview) {
	if a.previewChan == nil {
		panic("gui isn't initialized")
	}

	resultMonkeyChan := make(chan MonkeyPreview)
	monkeyPreviewChan = resultMonkeyChan
	go func() {
		select {
		case preview := <-resultMonkeyChan:
			a.previewChan <- preview
		case <-a.ctx.Done():
			return
		}
	}()
	return
}

func (a *MainApp) Levels() []logrus.Level {
	return []logrus.Level{logrus.PanicLevel, logrus.FatalLevel, logrus.ErrorLevel, logrus.WarnLevel, logrus.InfoLevel, logrus.DebugLevel}
}

func (a *MainApp) Fire(entry *logrus.Entry) error {
	go func() {
		a.logChan <- entry
	}()
	return nil
}

func (a *MainApp) processLogMessages() {
	for entry := range a.logChan {
		var msg string
		if entry.Level == logrus.InfoLevel {
			msg = entry.Message
		} else {
			msg = fmt.Sprintf("[%s]: %s", strings.ToUpper(entry.Level.String()), entry.Message)
		}

		a.app.QueueUpdateDraw(func() {
			fmt.Fprintln(a.logview, msg)
			a.logview.ScrollToEnd()
		}, a.logview)
	}
}

func NewMainApp(ctx context.Context, mainCancel context.CancelFunc, title string, log *logrus.Logger) *MainApp {
	mainApp := new(MainApp)
	mainApp.ctx, mainApp.mainCancel = ctx, mainCancel
	mainApp.logChan = make(chan *logrus.Entry, 5)
	log.AddHook(mainApp)
	go mainApp.processLogMessages()
	mainApp.previewChan = make(chan MonkeyPreview, 4)
	mainApp.statDeltas = make(chan engine.Stats, 20)
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

	logging := cview.NewTextView()
	flexBox.AddItem(logging, 0, 3, true)
	mainApp.logview = logging
	footer := cview.NewFlex()
	speed := cview.NewTextView()
	speed.SetTextAlign(cview.AlignRight)
	speed.SetText("engaged")
	mainApp.speed = speed
	total := cview.NewTextView()
	total.SetTextAlign(cview.AlignLeft)
	total.SetText("readying the hordes")
	mainApp.total = total
	footer.AddItem(total, 0, 3, false)
	footer.AddItem(speed, 0, 1, false)
	flexBox.AddItem(footer, 1, 0, false)
	mainApp.app.SetRoot(flexBox, true)
	captureHandler := func(event *tcell.EventKey) *tcell.EventKey {
		return mainApp.HandleEvent(event)
	}

	mainApp.app.SetInputCapture(captureHandler)

	mainApp.runtimeStats.Started = time.Now()
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

func (mainApp *MainApp) ForceQuit() {
	mainApp.app.QueueUpdate(func() { mainApp.app.Stop() })
}

func (m *MainApp) Quit() {
	// notify all users of the gui before shutting down the gui
	logrus.Debug("canceling main")
	m.mainCancel()
	// Cannot cleanup send channels from main until verification that main is done using them
main:
	for {
		deadline := time.NewTimer(time.Second * 10)
		select {
		case <-m.ctx.Done():
			logrus.Debug("Shutting down main gui as requested.")
			break main
		case <-deadline.C:
			logrus.Debug("tick")
		}

	}
	logrus.Debug("sending gui queue to stop qui")
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
			logrus.Debugf("Previewing %s", monkeyPreview.Title)
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

func (m *MainApp) Run(endTime time.Time) {
	m.endTime = endTime
	m.listenForPreviews()
	go func() {
		ticker := time.NewTicker(time.Second)
		for {
			<-ticker.C
			m.UpdateSpeed()
			m.UpdateTotal()
		}
	}()

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
