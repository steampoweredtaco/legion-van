package gui

import (
	"context"
	"fmt"
	"image"

	legion "github.com/steampoweredtaco/legion-van/image"

	"github.com/gdamore/tcell/v2"
	"github.com/gdamore/tcell/v2/views"
	"github.com/sirupsen/logrus"
)

type MainApp struct {
	root     MainAppRoot
	app      views.Application
	title    *views.Text
	status   *views.TextBar
	ctx      context.Context
	quit     context.CancelFunc
	preview1 *ImageModel
	preview2 *ImageModel
	preview3 *ImageModel
	preview4 *ImageModel
	log      *logrus.Logger
}

type MainAppRoot struct {
	views.BoxLayout
	mainApp *MainApp
}

type ImageModel struct {
	x           int
	y           int
	endx        int
	endy        int
	hide        bool
	loc         string
	enab        bool
	screen      tcell.SimulationScreen
	originalImg image.Image
}

func NewImageModel(img image.Image, viewWidth, viewHeight int) *ImageModel {
	model := &ImageModel{originalImg: img}
	model.screen = tcell.NewSimulationScreen("")
	model.resize(viewWidth, viewHeight)
	if img != nil {
		legion.DrawImgToScreen(model.screen, img)
	}
	return model
}

func (m *ImageModel) setImage(img image.Image) {
	m.originalImg = img
	m.screen.Clear()
	legion.DrawImgToScreen(m.screen, img)
}

func (m *ImageModel) resize(width, height int) {
	m.screen.SetSize(width, height)
}

func (m *ImageModel) GetBounds() (int, int) {
	return m.endx, m.endy
}

func (m *ImageModel) MoveCursor(offx, offy int) {
	m.x += offx
	m.y += offy
	m.limitCursor()
}

func (m *ImageModel) limitCursor() {
	if m.x < 0 {
		m.x = 0
	}
	if m.x > m.endx-1 {
		m.x = m.endx - 1
	}
	if m.y < 0 {
		m.y = 0
	}
	if m.y > m.endy-1 {
		m.y = m.endy - 1
	}
	m.loc = fmt.Sprintf("Cursor is %d,%d", m.x, m.y)
}

func (m *ImageModel) GetCursor() (int, int, bool, bool) {
	return m.x, m.y, m.enab, !m.hide
}

func (m *ImageModel) SetCursor(x int, y int) {
	m.x = x
	m.y = y

	m.limitCursor()
}

func (m *ImageModel) GetCell(x, y int) (rune, tcell.Style, []rune, int) {

	if m.screen == nil {
		return 0, tcell.StyleDefault, nil, 1
	}
	maxX, maxY := m.screen.Size()
	if x >= maxX || y >= maxY {
		return 0, tcell.StyleDefault, nil, 1
	}

	mainc, combc, style, width := m.screen.GetContent(x, y)
	return mainc, style, combc, width
}

func (a *MainApp) SetPreview1(img image.Image) {
	a.preview1.setImage(img)
}
func (a *MainApp) SetPreview2(img image.Image) {
	a.preview2.setImage(img)
}
func (a *MainApp) SetPreview3(img image.Image) {
	a.preview3.setImage(img)
}
func (a *MainApp) SetPreview4(img image.Image) {
	a.preview4.setImage(img)
}

func (a *MainApp) SetTerminalScreen(s tcell.Screen) {
	a.app.SetScreen(s)
}

func NewMainApp(title string, log *logrus.Logger) *MainApp {
	previewStyle := tcell.StyleDefault.Foreground(tcell.ColorGhostWhite)
	mainApp := new(MainApp)
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

	previewWindow1 := views.NewCellView()
	model := NewImageModel(nil, 30, 10)
	mainApp.preview1 = model
	previewWindow1.SetModel(model)
	previewWindow1.SetStyle(previewStyle)

	previewWindow2 := views.NewCellView()
	model = NewImageModel(nil, 30, 10)
	mainApp.preview2 = model
	previewWindow2.SetModel(model)
	previewWindow2.SetStyle(previewStyle)

	previewWindow3 := views.NewCellView()
	previewWindow3.SetStyle(previewStyle)
	model = NewImageModel(nil, 30, 10)
	mainApp.preview3 = model
	previewWindow3.SetModel(model)

	previewWindow4 := views.NewCellView()
	previewWindow4.SetStyle(previewStyle)
	model = NewImageModel(nil, 30, 10)
	mainApp.preview4 = model
	previewWindow4.SetModel(model)

	bodyMain.AddWidget(previewWindow1, .25)
	bodyMain.AddWidget(previewWindow2, .25)
	bodyMain.AddWidget(previewWindow3, .25)
	bodyMain.AddWidget(previewWindow4, .25)

	panel.SetTitle(mainApp.title)
	panel.SetContent(bodyMain)
	panel.SetStatus(mainApp.status)

	mainApp.root.AddWidget(panel, 1)
	mainApp.root.mainApp = mainApp
	mainApp.root.SetOrientation(views.Vertical)

	mainApp.app.SetRootWidget(&mainApp.root)
	mainApp.ctx, mainApp.quit = context.WithCancel(context.Background())

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
	// notify all users of the gui before shutting down the going
	m.quit()
	m.app.Quit()
}

func (m *MainApp) Run() {
	defer func() {
		if r := recover(); r != nil {
			m.app.Quit()
			m.log.Error(r)
		}
	}()

	m.app.Run()
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
