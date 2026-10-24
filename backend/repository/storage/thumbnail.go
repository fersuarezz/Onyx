package storage

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"onyx/domain"

	_ "image/gif"
	_ "image/png"

	xdraw "golang.org/x/image/draw"
)

const (
	thumbMaxDim    = 320
	thumbQuality   = 75
	thumbMaxPixels = 50_000_000
)

// lee la imagen de src y devuelve un io.Reader con su miniatura en
// JPEG, con el lado mayor reducido a thumbMaxDim y la proporción conservada.
func Thumbnail(src io.ReadSeeker) (io.Reader, error) {
	cfg, _, err := image.DecodeConfig(src)
	if err != nil {
		return nil, fmt.Errorf("%w: no es una imagen válida", domain.ErrInvalidInput)
	}

	if cfg.Width*cfg.Height > thumbMaxPixels {
		return nil, fmt.Errorf("%w: imagen demasiado grande (%dx%d)", domain.ErrInvalidInput, cfg.Width, cfg.Height)
	}

	if _, err := src.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("%w: rebobinar origen: %w", domain.ErrFileSystem, err)
	}

	img, _, err := image.Decode(src)
	if err != nil {
		return nil, fmt.Errorf("%w: no se pudo decodificar la imagen", domain.ErrInvalidInput)
	}

	bounds := img.Bounds()
	w, h := fitWithin(bounds.Dx(), bounds.Dy(), thumbMaxDim)

	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	xdraw.ApproxBiLinear.Scale(dst, dst.Bounds(), img, bounds, xdraw.Src, nil)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: thumbQuality}); err != nil {
		return nil, fmt.Errorf("%w: codificar thumbnail: %w", domain.ErrFileSystem, err)
	}

	return &buf, nil
}

// ajusta w y h para que el lado mayor sea max, conservando la
// proporción. Devuelve las medidas originales si ya caben.
func fitWithin(w, h, max_ int) (int, int) {
	if w <= max_ && h <= max_ {
		return w, h
	}

	if w > h {
		return max_, max(h*max_/w, 1)
	}

	return max(w*max_/h, 1), max_
}
