package image

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"io"
	"os"
	"syscall"
	"unsafe"

	"image/color"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"github.com/disintegration/imaging"
	termbox "github.com/gdamore/tcell/v2"
	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	log "github.com/sirupsen/logrus"
	"golang.org/x/image/font/gofont/goregular"
	"gopkg.in/gographics/imagick.v3/imagick"
)

const defaultRatio float64 = 3.0 / 7.0 // The terminal's default cursor width/height ratio

func rgb(c color.Color) (uint16, uint16, uint16) {
	r, g, b, _ := c.RGBA()
	// Reduce color values to the range [0, 15]
	return uint16(r >> 8), uint16(g >> 8), uint16(b >> 8)
}

// termColor converts a 24-bit RGB color into a term256 compatible approximation.
func termColor(r, g, b uint16) uint16 {
	rterm := (((r * 5) + 127) / 255) * 36
	gterm := (((g * 5) + 127) / 255) * 6
	bterm := (((b * 5) + 127) / 255)

	return rterm + gterm + bterm + 16 + 1 // termbox default color offset
}

// load an image stored in the given path
func load(filename string) (image.Image, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	img, _, err := image.Decode(file)
	return img, err
}

func getTermCharPixelWxH(screen termbox.Screen) (width, height int) {
	var size [4]uint16
	if _, _, err := syscall.Syscall6(syscall.SYS_IOCTL, uintptr(os.Stdout.Fd()), uintptr(syscall.TIOCGWINSZ), uintptr(unsafe.Pointer(&size)), 0, 0, 0); err != 0 {
	}
	width, height = int(size[2]), int(size[3])
	if width > 0 && height > 0 {
		log.Info("Got from term")
		return
	}
	widthScreen, heightScreen := screen.Size()
	// w/h = defaultRatio
	// w = h * defaultRatio
	// h = w/defaultRatio
	width = widthScreen * 3
	height = heightScreen * 7
	return
}

// scales calculates the image scale to fit within the terminal width/height
func scale(imgW, imgH, termW, termH int, whratio float64) float64 {
	hr := float64(imgH) / (float64(termH) * whratio)
	wr := float64(imgW) / float64(termW)
	return max(hr, wr, 1)
}

// imgArea calcuates the approximate rectangle a terminal cell takes up
func imgArea(termX, termY int, pixelsPerConsoleX, pixelsPerConsoleY float64) (int, int, int, int) {
	startX, startY := float64(termX)*pixelsPerConsoleX, float64(termY)*pixelsPerConsoleY
	endX, endY := startX+pixelsPerConsoleX, startY+pixelsPerConsoleY

	return int(startX), int(startY), int(endX), int(endY)
}

// avgRGB calculates the average RGB color within the given
// rectangle, and returns the [0,1] range of each component.
func avgRGB(img image.Image, startX, startY, endX, endY int) (int32, int32, int32) {
	var total = [3]int32{}
	var count int32
	for x := startX; x < endX; x++ {
		for y := startY; y < endY; y++ {
			if (!image.Point{x, y}.In(img.Bounds())) {
				continue
			}
			r, g, b := rgb(img.At(x, y))
			total[0] += int32(r)
			total[1] += int32(g)
			total[2] += int32(b)
			count++
		}
	}
	if count == 0 {
		return 0, 0, 0
	}

	r := total[0] / count
	g := total[1] / count
	b := total[2] / count
	return r, g, b
}

// max returns the maximum value
func max(values ...float64) float64 {
	var m float64
	for _, v := range values {
		if v > m {
			m = v
		}
	}
	return m
}

func DrawText(s termbox.Screen, x, y int, text string) {
	x2, y2 := s.Size()
	drawText(s, x, y, x2, y2, termbox.StyleDefault, text)
	s.Show()
}

func DrawTextBottom(s termbox.Screen, text string) {
	_, y := s.Size()
	DrawText(s, 0, y-1, text)
}
func drawText(s termbox.Screen, x1, y1, x2, y2 int, style termbox.Style, text string) {
	row := y1
	col := x1
	for _, r := range text {
		s.SetContent(col, row, r, nil, style)
		col++
		if col >= x2 {
			row++
			col = x1
		}
		if row > y2 {
			break
		}
	}
}

func DrawImgToScreen(screen termbox.Screen, img image.Image) {
	// Subtract one for another line to write to
	termWidth, termHeight := screen.Size()
	termPixWidth, termPixelHeight := getTermCharPixelWxH(screen)
	//termPixelRatio := float64(termPixWidth) / float64(termPixelHeight)
	//termRatio := float64(termWidth) / float64(termHeight)
	bounds := img.Bounds()

	// resizedImg := imaging.Fill(img, termPixelHeight, termPixWidth, imaging.Center, imaging.Lanczos)
	resizedImg := imaging.Fit(img, termPixelHeight, termPixWidth, imaging.Lanczos)

	bounds = resizedImg.Bounds()
	imgW, imgH := bounds.Dx(), bounds.Dy()
	//imgRatio := float64(imgW) / float64(imgH)

	pixelsPerConsoleX := float64(imgW) / float64(termWidth)
	pixelsPerConsoleY := float64(imgH) / float64(termHeight)

	screen.Clear()
	for y := 0; y < termHeight; y++ {
		for x := 0; x < termWidth; x++ {
			// Calculate average color for the corresponding image rectangle
			// fitting in this cell. We use a half-block trick, wherein the
			// lower half of the cell displays the character ▄, effectively
			// doubling the resolution of the canvas.
			startX, startY, endX, endY := imgArea(x, y, pixelsPerConsoleX, pixelsPerConsoleY)

			r, g, b := avgRGB(resizedImg, startX, startY, endX, (startY+endY)/2)
			r2, g2, b2 := avgRGB(resizedImg, startX, (startY+endY)/2, endX, endY)
			style := termbox.StyleDefault.
				Background(termbox.NewRGBColor(r, g, b)).
				Foreground(termbox.NewRGBColor(r2, g2, b2))
			screen.SetContent(x, y, '▄', nil, style)
		}
	}
	// Show is expected to be done by caller
}

func display(ctx context.Context, screen termbox.Screen, img image.Image) error {

	DrawImgToScreen(screen, img)

	for {

		select {
		case <-ctx.Done():
			return nil
		}
	}
}

func addLabel(img image.Image, x, y int, size float64, label string) image.Image {
	bounds := img.Bounds()
	newImg := image.NewRGBA(bounds)
	draw.Draw(newImg, newImg.Rect, img, bounds.Min, draw.Src)
	var white = color.RGBA{255, 0, 255, 255}

	col := image.NewUniform(white)
	ttF, err := truetype.Parse(goregular.TTF)
	if err != nil {
		log.Fatal("couldn't load fonts.")
	}
	ctx := freetype.NewContext()
	pt := freetype.Pt(x, y+int(ctx.PointToFixed(size)>>6))
	ctx.SetSrc(col)
	ctx.SetDst(newImg)
	ctx.SetFont(ttF)
	ctx.SetFontSize(size)
	ctx.SetClip(bounds)
	_, err = ctx.DrawString(label, pt)
	if err != nil {
		log.Fatal("could not draw title.")
	}
	//	glog.Infof("%v", p)
	return newImg
}

func DisplayImage(ctx context.Context, screen termbox.Screen, imgData io.Reader, title string) {

	data, err := io.ReadAll(imgData)
	if err != nil {
		log.Errorf("could not read monkey image data to display: %")
		return
	}
	imagePNG, err := ConvertSvgToBinary(data, PNGFormat, 250)
	if err != nil {
		log.Errorf("could not convert svg monkey image to png data to display: %s", err)
		return
	}

	img, _, err := image.Decode(bytes.NewReader(imagePNG))
	if err != nil {
		log.Errorf("could not decode image data to display: %s", err)
		return
	}
	display(ctx, screen, img)
}

const DefaultSize = 4000

func Init() {
	imagick.Initialize()
}
func Destroy() {
	imagick.Terminate()
}
func ConvertSvgToBinary(svgData []byte, format ImageFormat, size uint) ([]byte, error) {
	mw := imagick.NewMagickWand()
	mw.SetImageFormat("SVG")
	pixelWand := imagick.NewPixelWand()

	pixelWand.SetColor("none")
	mw.SetBackgroundColor(pixelWand)
	mw.SetImageUnits(imagick.RESOLUTION_PIXELS_PER_INCH)
	density := 96.0 * float64(size) / float64(DefaultSize)
	mw.SetResolution(density, density)
	err := mw.ReadImageBlob(svgData)
	if err != nil {
		return nil, fmt.Errorf("could not read svg: %w", err)
	}
	mw.SetImageCompression(imagick.COMPRESSION_NO)
	mw.SetImageCompressionQuality(100)
	mw.SetImageFormat(format.fmtUpper)
	mw.ResetIterator()
	return mw.GetImageBlob(), nil
}
