package video

import (
	"bytes"
	"image"
	"image/draw"
	"image/jpeg"

	_ "image/jpeg" // register JPEG decoder
	_ "image/png"  // register PNG decoder
)

// cropTo916 centre-crops an encoded image to a 9:16 aspect ratio and returns it
// as JPEG. Making the Veo first frame exactly 9:16 stops Veo from padding it
// with black bars. If decoding fails it returns the original bytes untouched.
func cropTo916(data []byte) []byte {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return data
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= 0 || h <= 0 {
		return data
	}

	// Target aspect 9:16 (width:height). Crop whichever dimension is too large.
	tw, th := w, h
	if w*16 > h*9 { // too wide
		tw = h * 9 / 16
	} else { // too tall
		th = w * 16 / 9
	}
	if tw <= 0 || th <= 0 {
		return data
	}

	x0 := b.Min.X + (w-tw)/2
	y0 := b.Min.Y + (h-th)/2
	dst := image.NewRGBA(image.Rect(0, 0, tw, th))
	draw.Draw(dst, dst.Bounds(), img, image.Pt(x0, y0), draw.Src)

	var out bytes.Buffer
	if err := jpeg.Encode(&out, dst, &jpeg.Options{Quality: 92}); err != nil {
		return data
	}
	return out.Bytes()
}
