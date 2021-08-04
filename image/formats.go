package image

type ImageFormat struct {
	fmt      string
	fmtUpper string
	fmtLower string
}

func (i ImageFormat) AsLower() string {
	return i.fmtLower
}

func (i ImageFormat) AsUpper() string {
	return i.fmtLower
}

var (
	PNGFormat = ImageFormat{"png", "PNG", "png"}
	SVGFormat = ImageFormat{"svg", "SVG", "svg"}
)
