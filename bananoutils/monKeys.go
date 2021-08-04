package bananoutils

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"

	legion "github.com/steampoweredtaco/legion-van/image"
)

func GrabMonkey(publicAddr Account, format legion.ImageFormat) (io.Reader, error) {
	var addressBuilder strings.Builder
	addressBuilder.WriteString("https://monkey.banano.cc/api/v1/monkey/")
	addressBuilder.WriteString(string(publicAddr))
	// svg is friendlier on the server, so do conversion if needed client side
	addressBuilder.WriteString("?format=svg")
	response, err := http.Get(addressBuilder.String())
	if err != nil {
		return nil, fmt.Errorf("could not get monkey %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return nil, fmt.Errorf("non 200 error returned (%d %s)", response.StatusCode, response.Status)
	}
	defer response.Body.Close()
	copy, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read from response buffer: %w", err)
	}
	if format == legion.SVGFormat {
		return bytes.NewBuffer(copy), nil
	}
	bs, err := legion.ConvertSvgToBinary(copy, format, 1000)
	if err != nil {
		return nil, fmt.Errorf("could not covert monkey %w", err)
	}
	return bytes.NewBuffer(bs), nil
}
