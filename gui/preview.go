package gui

import (
	"bytes"
	"context"
	"image"
	"io"
	"sync"

	log "github.com/sirupsen/logrus"
	"github.com/steampoweredtaco/legion-van/bananoutils"
	"github.com/steampoweredtaco/legion-van/engine"
	legionImage "github.com/steampoweredtaco/legion-van/image"
)

func PreviewMonkeys(ctx context.Context, previewChan chan<- MonkeyPreview, monkeyDataChan <-chan engine.MonkeyStats, wg *sync.WaitGroup) {
	defer wg.Done()
	if monkeyDataChan == nil {
		return
	}
	for {
		select {
		case monkey := <-monkeyDataChan:
			// start := time.Now()
			// grab as svg as it is nicer to the server and we can convert it locally
			monkeySVG, err := bananoutils.GrabMonkey(ctx, bananoutils.Account(monkey.PublicAddress), legionImage.SVGFormat)
			if err != nil {
				log.Warnf("could not convert monkey to preview: %s %s", monkey.SillyName, err)
				continue
			}
			data, err := io.ReadAll(monkeySVG)
			if err != nil {
				log.Warnf("could not read data for moneky to preview: %s %s", monkey.SillyName, err)
				continue
			}

			imagePNG, err := legionImage.ConvertSvgToBinary(data, legionImage.PNGFormat, 250)
			if err != nil {
				log.Warnf("could not convert svg monkey image to png data to display: %s %s", monkey.SillyName, err)
				continue
			}

			img, _, err := image.Decode(bytes.NewReader(imagePNG))
			if err != nil {
				log.Warnf("could not decode image data to display for monkey: %s %s", monkey.SillyName, err)
				continue
			}
			select {
			case previewChan <- MonkeyPreview{Image: img, Title: monkey.SillyName}:
			case <-ctx.Done():
			}

		case <-ctx.Done():
			log.Debugf("stopping the monKey preview")
			return
		}
	}

}
