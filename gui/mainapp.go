/*
gui package contains the main vanity gui interface that provides convinence and channels to windows.
*/
package gui

import (
	"context"
	"image"
	"sync"

	"github.com/gdamore/tcell/v2"
	"github.com/gdamore/tcell/v2/views"
	"github.com/sirupsen/logrus"
)

type MonkeyPreview struct {
	Image image.Image
	Title string
}

type previewModel struct {
	imageViewModel *imageModel
	titleWidget    *views.Text
}

type MainApp struct {
	root        MainAppRoot
	app         views.Application
	title       *views.Text
	status      *views.TextBar
	ctx         context.Context
	mainCancel  context.CancelFunc
	preview     [4]*previewModel
	previewChan chan MonkeyPreview
	log         *logrus.Logger

	previewWG *sync.WaitGroup
}

type MainAppRoot struct {
	views.BoxLayout
	mainApp *MainApp
}

func (a *MainApp) SetPreview(index int, img image.Image, title string) {
	a.preview[index].imageViewModel.setImage(img)
	a.preview[index].titleWidget.SetText(title)
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

	previewStyle := tcell.StyleDefault.Foreground(tcell.ColorGhostWhite)
	mainApp := new(MainApp)
	mainApp.previewChan = make(chan MonkeyPreview, 4)

	if log != nil {
		mainApp.log = log
	}
	panel := views.NewPanel()
	mainApp.status = views.NewTextBar()
	mainApp.status.SetLeft("Simple", tcell.StyleDefault.Background(tcell.ColorYellowGreen))
	mainApp.title = views.NewText()
	mainApp.title.SetText(title)
	mainApp.title.SetAlignment(views.HAlignCenter)
	body := views.NewBoxLayout(views.Vertical)
	body.SetOrientation(views.Vertical)
	bodyMain := views.NewBoxLayout(views.Horizontal)

	for i := 0; i < 4; i++ {
		previewPanel := views.NewPanel()
		title := views.NewText()
		title.SetAlignment(views.HAlignCenter)
		previewPanel.SetTitle(title)
		previewWindow := views.NewCellView()
		model := NewImageModel(nil, 30, 10)
		mainApp.preview[i] = &previewModel{imageViewModel: model, titleWidget: title}
		previewPanel.Watch(model)
		previewPanel.AddWidget(previewWindow, .25)
		bodyMain.AddWidget(previewPanel, .25)
		previewWindow.SetModel(model)
		previewWindow.SetStyle(previewStyle)
	}
	panel.SetTitle(mainApp.title)
	panel.SetContent(bodyMain)
	panel.SetStatus(mainApp.status)

	mainApp.root.AddWidget(panel, 1)
	mainApp.root.mainApp = mainApp
	mainApp.root.SetOrientation(views.Vertical)

	mainApp.app.SetRootWidget(&mainApp.root)
	mainApp.ctx, mainApp.mainCancel = ctx, mainCancel
	return mainApp
}

func (mainApp *MainApp) Done() <-chan struct{} {
	return mainApp.ctx.Done()
}

func (mainApp *MainApp) ForceQuit() {
	mainApp.app.Quit()
}

func (m *MainApp) Quit() {
	defer func() {
		if r := recover(); r != nil {
			m.app.Quit()
			m.log.Error(r)
		}
	}()
	// notify all users of the gui before shutting down the gui
	m.mainCancel()
	m.app.Quit()
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
			previewModel := m.preview[nextPreviewPane%4]
			previewModel.imageViewModel.setImage(monkeyPreview.Image)
			previewModel.titleWidget.SetText(monkeyPreview.Title)
			nextPreviewPane++
			m.app.Update()
		}
	}
}

func (m *MainApp) listenForPreviews() {
	m.previewWG = new(sync.WaitGroup)
	// TODO make the number of preview panes configurable.
	m.previewWG.Add(4)
	go m.processPreviews()
}

func (m *MainApp) cleanupGui() {
	m.previewWG.Wait()
}

func (m *MainApp) Run() {
	defer func() {
		if r := recover(); r != nil {
			m.app.Quit()
			m.log.Error(r)
		}
	}()

	m.listenForPreviews()
	m.app.Run()
	m.cleanupGui()
}

func (root *MainAppRoot) HandleEvent(event tcell.Event) bool {
	switch ev := event.(type) {
	case *tcell.EventKey:
		if ev.Key() == tcell.KeyEscape {
			root.mainApp.Quit()
			return true
		}
	}
	return root.BoxLayout.HandleEvent(event)
}
