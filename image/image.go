package image

import (
	"image"
	"io"
	"os"
	"syscall"
	"time"
	"unsafe"

	"image/color"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"github.com/golang/glog"
	"github.com/mattn/go-runewidth"
	"github.com/nsf/termbox-go"
	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
	"golang.org/x/image/font/gofont/goregular"
)

const defaultRatio float64 = 7.0 / 3.0 // The terminal's default cursor width/height ratio

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

// canvasSize returns the terminal columns, rows, and cursor aspect ratio
func canvasSize() (int, int, float64) {
	var size [4]uint16
	if _, _, err := syscall.Syscall6(syscall.SYS_IOCTL, uintptr(os.Stdout.Fd()), uintptr(syscall.TIOCGWINSZ), uintptr(unsafe.Pointer(&size)), 0, 0, 0); err != 0 {
		panic(err)
	}
	rows, cols, width, height := size[0], size[1], size[2], size[3]
	var whratio = defaultRatio
	if width > 0 && height > 0 {
		whratio = float64(height/rows) / float64(width/cols)
	}

	return int(cols), int(rows), whratio
}

// scales calculates the image scale to fit within the terminal width/height
func scale(imgW, imgH, termW, termH int, whratio float64) float64 {
	hr := float64(imgH) / (float64(termH) * whratio)
	wr := float64(imgW) / float64(termW)
	return max(hr, wr, 1)
}

// imgArea calcuates the approximate rectangle a terminal cell takes up
func imgArea(termX, termY int, imgScale, whratio float64) (int, int, int, int) {
	startX, startY := float64(termX)*imgScale, float64(termY)*imgScale*whratio
	endX, endY := startX+imgScale, startY+imgScale*whratio

	return int(startX), int(startY), int(endX), int(endY)
}

// avgRGB calculates the average RGB color within the given
// rectangle, and returns the [0,1] range of each component.
func avgRGB(img image.Image, startX, startY, endX, endY int) (uint16, uint16, uint16) {
	var total = [3]uint16{}
	var count uint16
	for x := startX; x < endX; x++ {
		for y := startY; y < endY; y++ {
			if (!image.Point{x, y}.In(img.Bounds())) {
				continue
			}
			r, g, b := rgb(img.At(x, y))
			total[0] += r
			total[1] += g
			total[2] += b
			count++
		}
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

func tbprint(x, y int, fg, bg termbox.Attribute, msg string) {
	originalX := x
	for _, c := range msg {
		if c == '\n' {
			x = originalX
			y++
			continue
		}
		termbox.SetCell(x, y, c, fg, bg)
		x += runewidth.RuneWidth(c)
	}
}

func drawImgToTerminal(img image.Image, title string) {
	// Get terminal size and cursor width/height ratio
	width, height, whratio := canvasSize()
	// Subtract one for another line to write to
	bounds := img.Bounds()
	imgW, imgH := bounds.Dx(), bounds.Dy()

	imgScale := scale(imgW, imgH, width, height, whratio)

	// Resize canvas to fit scaled image
	width, height = int(float64(imgW)/imgScale), int(float64(imgH)/(imgScale*whratio))

	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
	white := termbox.ColorWhite
	black := termbox.ColorBlack

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// Calculate average color for the corresponding image rectangle
			// fitting in this cell. We use a half-block trick, wherein the
			// lower half of the cell displays the character ▄, effectively
			// doubling the resolution of the canvas.
			startX, startY, endX, endY := imgArea(x, y, imgScale, whratio)

			r, g, b := avgRGB(img, startX, startY, endX, (startY+endY)/2)
			colorUp := termbox.Attribute(termColor(r, g, b))

			r, g, b = avgRGB(img, startX, (startY+endY)/2, endX, endY)
			colorDown := termbox.Attribute(termColor(r, g, b))

			termbox.SetCell(x, y, '▄', colorDown, colorUp)
		}
	}
	tbprint(0, 0, white, black, title)
	termbox.Flush()
}

func display(img image.Image, title string) error {
	drawImgToTerminal(img, title)

	for {
		switch ev := termbox.PollEvent(); ev.Type {
		case termbox.EventKey:
			return nil
		case termbox.EventResize:
			drawImgToTerminal(img, title)
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func DisplaySVG(imgData io.Reader, title string) {
	w, h := 512, 512

	icon, _ := oksvg.ReadIconStream(imgData)
	icon.SetTarget(0, 0, float64(w), float64(h))
	rgba := image.NewRGBA(image.Rect(0, 0, w, h))
	icon.Draw(rasterx.NewDasher(w, h, rasterx.NewScannerGV(w, h, rgba, rgba.Bounds())), 1)

	err := termbox.Init()
	if err != nil {
		glog.Fatalf("could not int termbox %v", err)
	}
	defer termbox.Close()
	termbox.SetOutputMode(termbox.Output256)

	display(rgba, title)
}
func addLabel(img image.Image, x, y int, size float64, label string) image.Image {
	bounds := img.Bounds()
	newImg := image.NewRGBA(bounds)
	draw.Draw(newImg, newImg.Rect, img, bounds.Min, draw.Src)
	var white = color.RGBA{255, 0, 255, 255}

	col := image.NewUniform(white)
	ttF, err := truetype.Parse(goregular.TTF)
	if err != nil {
		glog.Fatal("couldn't load fonts.")
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
		glog.Fatal("could not draw title.")
	}
	//	glog.Infof("%v", p)
	return newImg
}

func DisplayImage(imgData io.Reader, title string) {
	glog.Infoln("Close the image with <ESC> or by pressing 'q'.")

	err := termbox.Init()
	if err != nil {
		glog.Fatalf("could not int termbox %v", err)
	}
	defer termbox.Close()
	termbox.SetOutputMode(termbox.Output256)
	img, _, err := image.Decode(imgData)
	if err != nil {
		glog.Errorf("could not decode image data")
	}
	//panic("doh")
	display(img, title)
}
