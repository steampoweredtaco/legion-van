/*
gui package contains the main vanity gui interface that provides convinence and channels to windows.
*/
package gui

import (
	"context"
	"image"
	"sync"
	"time"

	"code.rocketnine.space/tslocum/cview"
	"github.com/gdamore/tcell/v2"
	"github.com/gdamore/tcell/v2/views"
	"github.com/sirupsen/logrus"
)

type MonkeyPreview struct {
	Image image.Image
	Title string
}

type MainApp struct {
	app          *cview.Application
	status       *views.TextBar
	ctx          context.Context
	mainCancel   context.CancelFunc
	preview      [4]*imageBox
	previewChan  chan MonkeyPreview
	log          *logrus.Logger
	previewWG    *sync.WaitGroup
	mainDoneChan chan struct{}
	once         sync.Once
}

func (a *MainApp) SetPreview(index int, img image.Image, title string) {
	a.preview[index].setImage(img)
	a.preview[index].SetTitle(title)
	a.preview[index].SetTitleAlign(cview.AlignCenter)
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
	if log != nil {
		mainApp.log = log
	}

	mainApp.app = cview.NewApplication()

	flexBox := cview.NewFlex()
	flexBox.SetFullScreen(true)
	flexBox.SetTitle(title)
	for i := 0; i < 4; i++ {
		imgBox := NewImageBox(nil, 30, 10)
		imgBox.SetBorder(true)
		flexBox.AddItem(imgBox, 0, 1, false)
		mainApp.preview[i] = imgBox
	}

	mainApp.app.SetRoot(flexBox, true)
	captureHandler := func(event *tcell.EventKey) *tcell.EventKey {
		return mainApp.HandleEvent(event)
	}

	mainApp.app.SetInputCapture(captureHandler)

	mainApp.ctx, mainApp.mainCancel = ctx, mainCancel
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
	mainApp.mainDoneChan <- struct{}{}
}

func (mainApp *MainApp) ForceQuit() {
	mainApp.app.QueueUpdate(func() { mainApp.app.Stop() })
}

func (m *MainApp) Quit() {
	// notify all users of the gui before shutting down the gui
	m.mainCancel()
	// Cannot cleanup send channels from main until verification that main is done using them
main:
	for {
		deadline := time.NewTimer(time.Second * 10)
		select {
		case <-m.mainDoneChan:
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
	logrus.Info("Set hasShutdown to true")
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
			m.app.QueueUpdateDraw(func() {}, imageBox)
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
	m.app.Run()
	m.cleanupGui()
}

func (mainApp *MainApp) HandleEvent(event *tcell.EventKey) *tcell.EventKey {
	if event.Key() == tcell.KeyEscape {
		mainApp.once.Do(mainApp.Quit)
		return nil
	}
	return event
}
