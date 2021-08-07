package gui

import (
	"image"
	"math"

	"code.rocketnine.space/tslocum/cview"
	"github.com/gdamore/tcell/v2"
	legion "github.com/steampoweredtaco/legion-van/image"
)

const (
	// width/height of a standard console character
	defaultConsolePicRation = float64(3 / 7)
)

type imageBox struct {
	cview.Box
	screen tcell.SimulationScreen
	img    image.Image
	aspect float64
	width  int
	height int
}

func (b *imageBox) imageBoxDrawFunc(screen tcell.Screen, topLeftX, topLeftY, width, height int) (int, int, int, int) {
	// getting the border while in this function will cause cview to deadlock, so assume always a border
	borderAdjustment := 1

	b.resize(width-borderAdjustment*2, height-borderAdjustment*2)
	if b.img != nil {
		legion.DrawImgToScreen(b.screen, b.img)
	}
	for y := 0 + borderAdjustment; y < b.height-borderAdjustment; y++ {
		for x := 0 + borderAdjustment; x < b.width-borderAdjustment; x++ {
			rune, comb, style, wid := b.screen.GetContent(x, y)
			// Box widget takes care of nothing in a good way
			if rune == 0 {
				continue
			}
			screen.SetContent(topLeftX+x, topLeftY+y, rune, comb, style)
			x += wid - 1
		}
	}
	return topLeftX, topLeftY, width, height
}

func (b *imageBox) getImageDrawFunc() func(screen tcell.Screen, topLeftX, topLeftY, width, height int) (int, int, int, int) {
	return func(screen tcell.Screen, topLeftX, topLeftY, width, height int) (int, int, int, int) {
		return b.imageBoxDrawFunc(screen, topLeftX, topLeftY, width, height)
	}
}

func NewImageBox(img image.Image, viewWidth, viewHeight int) *imageBox {
	box := &imageBox{img: img}
	box.Box = *cview.NewBox()
	box.screen = tcell.NewSimulationScreen("")
	box.aspect = float64(viewWidth) / float64(viewHeight)
	box.resize(viewWidth, viewHeight)
	if img != nil {
		legion.DrawImgToScreen(box.screen, img)
	}
	box.Box.SetDrawFunc(box.getImageDrawFunc())
	return box
}

func (b *imageBox) resize(width, height int) {
	var adjustedHeight float64
	var adjustedWidth float64
	if b.aspect > 1 {
		// capped at width because it is larger in the original requested image
		adjustedWidth = float64(width)
		adjustedHeight = math.Ceil(float64(width) / b.aspect)
		// can't fit so need to adjust
		if adjustedHeight > float64(height) {
			adjustedHeight = float64(height)
			adjustedWidth = float64(adjustedHeight) * b.aspect
		}
	} else {
		adjustedHeight = float64(height)
		adjustedWidth = float64(height) * b.aspect
		if adjustedWidth > float64(width) {
			adjustedWidth = float64(width)
			adjustedHeight = math.Ceil(float64(width) / b.aspect)
		}

	}
	b.width = int(adjustedWidth)
	b.height = int(adjustedHeight)
	b.screen.SetSize(b.width, b.height)
}

func (b *imageBox) setImage(img image.Image) {
	b.img = img
	b.screen.Clear()
	legion.DrawImgToScreen(b.screen, img)
}
