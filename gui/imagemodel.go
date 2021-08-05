package gui

import (
	"fmt"
	"image"

	"github.com/gdamore/tcell/v2"
	"github.com/gdamore/tcell/v2/views"
	log "github.com/sirupsen/logrus"
	legion "github.com/steampoweredtaco/legion-van/image"
)

type imageModel struct {
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

func NewImageModel(img image.Image, viewWidth, viewHeight int) *imageModel {
	model := &imageModel{originalImg: img}
	model.screen = tcell.NewSimulationScreen("")
	model.resize(viewWidth, viewHeight)
	if img != nil {
		legion.DrawImgToScreen(model.screen, img)
	}
	return model
}

func (m *imageModel) setImage(img image.Image) {
	m.originalImg = img
	m.screen.Clear()
	legion.DrawImgToScreen(m.screen, img)
}

func (m *imageModel) HandleEvent(event tcell.Event) bool {
	log.Printf("heard event %T", event)
	switch ev := event.(type) {
	case *views.EventWidgetResize:
		w, h := ev.Widget().Size()
		m.resize(w, h)
		return true
	}
	return false
}
func (m *imageModel) resize(width, height int) {
	log.WithField("width", width).WithField("height", height).Logger.Info("resized")
	m.screen.SetSize(width, height)
}

func (m *imageModel) GetBounds() (int, int) {
	return m.endx, m.endy
}

func (m *imageModel) MoveCursor(offx, offy int) {
	m.x += offx
	m.y += offy
	m.limitCursor()
}

func (m *imageModel) limitCursor() {
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

func (m *imageModel) GetCursor() (int, int, bool, bool) {
	return m.x, m.y, m.enab, !m.hide
}

func (m *imageModel) SetCursor(x int, y int) {
	m.x = x
	m.y = y

	m.limitCursor()
}

func (m *imageModel) GetCell(x, y int) (rune, tcell.Style, []rune, int) {

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
