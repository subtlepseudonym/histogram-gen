package main

import (
	// "fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"log"
	"math"
	"os"
)

const (
	// TODO: take from command line arg
	// TODO: use flag for log scale height
	histHeight uint = 226 // pixels
	histWidth  uint = 2200
	// TODO: fix broken behavior when histWidth > paletteWidth
	paletteWidth uint32 = 0xffff + 1 // size of uint16
)

var (
	black   = color.RGBA{0, 0, 0, 0xff}
	red     = color.RGBA{0xff, 0, 0, 0xff}
	green   = color.RGBA{0, 0xff, 0, 0xff}
	blue    = color.RGBA{0, 0, 0xff, 0xff}
	yellow  = color.RGBA{0xff, 0xff, 0, 0xff}
	magenta = color.RGBA{0xff, 0, 0xff, 0xff}
	cyan    = color.RGBA{0, 0xff, 0xff, 0xff}
	// TODO: add flag for white/gray as neutral color
	white  = color.RGBA{0xff, 0xff, 0xff, 0xff}
	gray   = color.RGBA{0xE0, 0xE0, 0xE0, 0xff} // good for white backgrounds
	errLog = log.New(os.Stderr, "ERROR: ", log.LstdFlags)
	// outLog  = log.New(os.Stdout, "", 7)
)

func decodeJpeg(filepath string) image.Image {
	pic, err := os.Open(filepath)
	if err != nil {
		errLog.Print("Failed to open image file")
		panic(err)
	}

	jpgImage, err := jpeg.Decode(pic)
	if err != nil {
		errLog.Print("Failed to decode image file, expected .jpeg or .jpg")
		panic(err)
	}

	return jpgImage
}

func getHistColorBoundaries(colorVals [3][65536]uint, widthCoeff float64) ([histWidth][3]uint, uint) {
	var histDesc [histWidth][3]uint
	var maxDesc uint

	for i := 0; i < int(histWidth); i++ {
		var avgR, avgG, avgB float64 // will be converted to uint when inserted into histDesc

		// Account for latter half of values only partially averaged into previous histogram bar
		if i-1 > 0 {
			avgR += float64(colorVals[0][(i-1)*int(widthCoeff)]) * (1. - (math.Mod(widthCoeff, 1.)))
			avgG += float64(colorVals[1][(i-1)*int(widthCoeff)]) * (1. - (math.Mod(widthCoeff, 1.)))
			avgB += float64(colorVals[2][(i-1)*int(widthCoeff)]) * (1. - (math.Mod(widthCoeff, 1.)))
		}

		// Add rgb values to running total
		for j := 0; j < int(widthCoeff); j++ {
			avgR += float64(colorVals[0][j+int(widthCoeff)*i])
			avgG += float64(colorVals[1][j+int(widthCoeff)*i])
			avgB += float64(colorVals[2][j+int(widthCoeff)*i])
		}

		// Add fractional part of bar
		avgR += float64(colorVals[0][int(widthCoeff)*(i+1)]) * (math.Mod(widthCoeff, 1))
		avgG += float64(colorVals[1][int(widthCoeff)*(i+1)]) * (math.Mod(widthCoeff, 1))
		avgB += float64(colorVals[2][int(widthCoeff)*(i+1)]) * (math.Mod(widthCoeff, 1))

		// Turn into actual averages of rgb values along bar with width of value widthCoeff
		avgR /= widthCoeff
		avgG /= widthCoeff
		avgB /= widthCoeff

		// Scale averages to histHeight and store
		histDesc[i][0] = uint(avgR) // / heightDenom)
		histDesc[i][1] = uint(avgG) // / heightDenom)
		histDesc[i][2] = uint(avgB) // / heightDenom)

		if histDesc[i][0] > maxDesc {
			maxDesc = histDesc[i][0]
		}
		if histDesc[i][1] > maxDesc {
			maxDesc = histDesc[i][1]
		}
		if histDesc[i][2] > maxDesc {
			maxDesc = histDesc[i][2]
		}
	}

	return histDesc, maxDesc
}

func drawBgColor(idx int, valHeight uint, destImg draw.Image) {
	draw.Draw(destImg, image.Rect(idx, 0, idx+1, int(histHeight-valHeight)), &image.Uniform{black}, image.Pt(idx, 0), draw.Src)
}

func drawFgColor(idx int, topVal, midVal, lowVal uint, topColor, midColor color.RGBA, destImg draw.Image) {
	draw.Draw(destImg, image.Rect(idx, int(histHeight-topVal), idx+1, int(histHeight-midVal)), &image.Uniform{topColor}, image.Pt(idx, int(topVal)), draw.Src)
	draw.Draw(destImg, image.Rect(idx, int(histHeight-midVal), idx+1, int(histHeight-lowVal)), &image.Uniform{midColor}, image.Pt(idx, int(midVal)), draw.Src)
	draw.Draw(destImg, image.Rect(idx, int(histHeight-lowVal), idx+1, int(histHeight)), &image.Uniform{gray}, image.Pt(idx, int(lowVal)), draw.Src)
}

func drawHistogram(histDesc [histWidth][3]uint, norm float64, histogram draw.Image) draw.Image {
	for i := 0; i < int(histWidth); i++ {
		r := uint(float64(histDesc[i][0]) * norm)
		g := uint(float64(histDesc[i][1]) * norm)
		b := uint(float64(histDesc[i][2]) * norm)

		// Hideous control flow
		// Also slightly optimised for mostly red and yellow images
		if r >= g && r >= b { // r has highest value
			drawBgColor(i, r, histogram)
			if g >= b { // g has second highest value
				drawFgColor(i, r, g, b, red, yellow, histogram)
			} else {
				drawFgColor(i, r, b, g, red, magenta, histogram)
			}
		} else if g >= b && g >= r { // g has highest value
			drawBgColor(i, g, histogram)
			if r >= b { // r has second highest value
				drawFgColor(i, g, r, b, green, yellow, histogram)
			} else {
				drawFgColor(i, g, b, r, green, cyan, histogram)
			}
		} else { // b has highest value
			drawBgColor(i, b, histogram)
			if r >= g { // r has second highest value
				drawFgColor(i, b, r, g, blue, magenta, histogram)
			} else {
				drawFgColor(i, b, g, r, blue, cyan, histogram)
			}
		}
	}

	return histogram
}

func main() {
	args := os.Args[1:]
	jpgImage := decodeJpeg(args[0])

	// FIXME: uint is probably 64bit and may overflow / cause errors later with uint32's
	var colorVals [3][paletteWidth]uint
	for i := 0; i < jpgImage.Bounds().Max.X-1; i++ {
		for j := 0; j < jpgImage.Bounds().Max.Y-1; j++ {
			r, g, b, _ := jpgImage.At(i, j).RGBA()
			colorVals[0][r]++
			colorVals[1][g]++
			colorVals[2][b]++
		}
	}

	widthCoeff := float64(paletteWidth) / float64(histWidth) // number of rgb vals to use for each bar
	histDesc, maxDesc := getHistColorBoundaries(colorVals, widthCoeff)

	norm := float64(histHeight) * 0.95 / float64(maxDesc)
	histogram := drawHistogram(histDesc, norm, image.NewRGBA(image.Rect(0, 0, int(histWidth), int(histHeight))))

	file, _ := os.OpenFile("./tmp2.jpg", os.O_WRONLY, os.ModeAppend)
	defer file.Close()
	err := jpeg.Encode(file, histogram, &jpeg.Options{100})
	if err != nil {
		panic(err)
	}
}
