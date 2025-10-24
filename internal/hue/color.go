package hue

import (
	"math"
)

var (
	defaultGamut = Gamut{
		Red:   Coords{X: 0.6915, Y: 0.3083},
		Green: Coords{X: 0.1700, Y: 0.7000},
		Blue:  Coords{X: 0.1532, Y: 0.0475},
	}

	gamutMap = map[GamutType]Gamut{
		GamutTypeA: {
			Red:   Coords{X: 0.704, Y: 0.296},
			Green: Coords{X: 0.2151, Y: 0.7106},
			Blue:  Coords{X: 0.138, Y: 0.08},
		},
		GamutTypeB: {
			Red:   Coords{X: 0.675, Y: 0.322},
			Green: Coords{X: 0.409, Y: 0.518},
			Blue:  Coords{X: 0.167, Y: 0.04},
		},
		GamutTypeC: {
			Red:   Coords{X: 0.6915, Y: 0.3083},
			Green: Coords{X: 0.1700, Y: 0.7000},
			Blue:  Coords{X: 0.1532, Y: 0.0475},
		},
	}
)

// ColorToRGB converts a Light to RGB with gamut correction
func ColorToRGB(light *Light, fixedBrightness *int) (int, int, int) {
	if !light.On.On {
		return 0, 0, 0
	}

	brightness := int(light.Dimming.Brightness)
	if fixedBrightness != nil {
		brightness = *fixedBrightness
	}

	if light.ColorTemperature.MirekValid {
		// CT is in mireds, convert to Kelvin: 1000000/CT
		kelvin := 1000000 / light.ColorTemperature.Mirek
		return ctToRGB(kelvin, brightness)
	}

	if light.Color.XY.X != 0 || light.Color.XY.Y != 0 {
		return coordsToRGB(
			light.Color.XY.X,
			light.Color.XY.Y,
			brightness,
			light.Color.GamutType,
			light.Color.Gamut,
		)
	}

	brightnessValue := brightness * 255 / 100
	return brightnessValue, brightnessValue, brightnessValue
}

// coordsToRGB converts XY coordinates and brightness to RGB with gamut correction
func coordsToRGB(x, y float64, bri int, gamutType GamutType, gamut Gamut) (int, int, int) {
	var colorGamut Gamut
	if isValidGamut(gamut) {
		colorGamut = gamut
	} else if g, ok := gamutMap[gamutType]; ok {
		colorGamut = g
	} else {
		colorGamut = defaultGamut
	}

	correctedCoords := correctToGamut(Coords{X: x, Y: y}, colorGamut)
	x, y = correctedCoords.X, correctedCoords.Y

	z := 1.0 - x - y
	Y := float64(bri) / 100.0 // Brightness is 0-100 in the new API
	X := (Y / y) * x
	Z := (Y / y) * z

	// convert to linear sRGB
	rLin := X*3.2406 + Y*-1.5372 + Z*-0.4986
	gLin := X*-0.9689 + Y*1.8758 + Z*0.0415
	bLin := X*0.0557 + Y*-0.2040 + Z*1.0570

	toGamma := func(v float64) float64 {
		if v <= 0.0031308 {
			return 12.92 * v
		}
		return 1.055*math.Pow(v, 1.0/2.4) - 0.055
	}
	r, g, b := int(clamp(toGamma(rLin)*255, 0, 255)),
		int(clamp(toGamma(gLin)*255, 0, 255)),
		int(clamp(toGamma(bLin)*255, 0, 255))

	return r, g, b
}

// isValidGamut checks if a gamut has valid coordinates
func isValidGamut(gamut Gamut) bool {
	return !(gamut.Red.X == 0 && gamut.Red.Y == 0 &&
		gamut.Green.X == 0 && gamut.Green.Y == 0 &&
		gamut.Blue.X == 0 && gamut.Blue.Y == 0)
}

// ctToRGB converts color temperature in Kelvin to RGB
func ctToRGB(kelvin int, bri int) (int, int, int) {
	// Algorithm based on https://tannerhelland.com/2012/09/18/convert-temperature-rgb-algorithm-code.html
	temp := float64(kelvin) / 100.0

	var r, g, b float64

	// Calculate red
	if temp <= 66 {
		r = 255
	} else {
		r = temp - 60
		r = 329.698727446 * math.Pow(r, -0.1332047592)
		if r < 0 {
			r = 0
		}
		if r > 255 {
			r = 255
		}
	}

	// Calculate green
	if temp <= 66 {
		g = temp
		g = 99.4708025861*math.Log(g) - 161.1195681661
	} else {
		g = temp - 60
		g = 288.1221695283 * math.Pow(g, -0.0755148492)
	}
	if g < 0 {
		g = 0
	}
	if g > 255 {
		g = 255
	}

	// Calculate blue
	if temp >= 66 {
		b = 255
	} else if temp <= 19 {
		b = 0
	} else {
		b = temp - 10
		b = 138.5177312231*math.Log(b) - 305.0447927307
		if b < 0 {
			b = 0
		}
		if b > 255 {
			b = 255
		}
	}

	// Apply brightness
	brightness := float64(bri) / 100.0
	r = r * brightness
	g = g * brightness
	b = b * brightness

	return int(r), int(g), int(b)
}

// isInGamut checks if point is inside triangle
func isInGamut(point Coords, gamut Gamut) bool {
	v1x, v1y := gamut.Red.X, gamut.Red.Y
	v2x, v2y := gamut.Green.X, gamut.Green.Y
	v3x, v3y := gamut.Blue.X, gamut.Blue.Y

	denominator := (v2y-v3y)*(v1x-v3x) + (v3x-v2x)*(v1y-v3y)
	a := ((v2y-v3y)*(point.X-v3x) + (v3x-v2x)*(point.Y-v3y)) / denominator
	b := ((v3y-v1y)*(point.X-v3x) + (v1x-v3x)*(point.Y-v3y)) / denominator
	c := 1 - a - b

	return a >= 0 && a <= 1 && b >= 0 && b <= 1 && c >= 0 && c <= 1
}

// correctToGamut projects point to gamut edge if outside the gamut
func correctToGamut(point Coords, gamut Gamut) Coords {
	if isInGamut(point, gamut) {
		return point
	}

	closestDist := 1e10
	closestPoint := point

	findClosestPoint := func(p1, p2, target Coords) (Coords, float64) {
		dx := p2.X - p1.X
		dy := p2.Y - p1.Y

		lenSq := dx*dx + dy*dy

		// If segment is just a point, return distance to that point
		if lenSq < 1e-10 {
			dist := math.Sqrt((target.X-p1.X)*(target.X-p1.X) + (target.Y-p1.Y)*(target.Y-p1.Y))
			return p1, dist
		}

		t := ((target.X-p1.X)*dx + (target.Y-p1.Y)*dy) / lenSq
		t = math.Max(0, math.Min(1, t))

		// Closest point on segment
		projX := p1.X + t*dx
		projY := p1.Y + t*dy

		dist := math.Sqrt((target.X-projX)*(target.X-projX) + (target.Y-projY)*(target.Y-projY))
		return Coords{X: projX, Y: projY}, dist
	}

	edges := []struct {
		p1, p2 Coords
	}{
		{gamut.Red, gamut.Green},
		{gamut.Green, gamut.Blue},
		{gamut.Blue, gamut.Red},
	}

	for _, edge := range edges {
		projPoint, dist := findClosestPoint(edge.p1, edge.p2, point)

		if dist < closestDist {
			closestDist = dist
			closestPoint = projPoint
		}
	}

	return closestPoint
}

func clamp(x, minF, maxF float64) float64 {
	if x < minF {
		return minF
	}
	if x > maxF {
		return maxF
	}
	return x
}
