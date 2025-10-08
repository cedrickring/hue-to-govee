package hue

import (
	"context"
	"sync"
	"time"

	"github.com/cedrickring/hue-to-govee/internal/govee"
	"github.com/rs/zerolog"
)

// SceneController manages dynamic scenes for Govee devices
type SceneController struct {
	mu           sync.Mutex // Mutex to protect activeScenes updates
	activeScenes map[string]context.CancelFunc

	logger      zerolog.Logger
	goveeClient *govee.Client
}

// NewSceneController creates a new SceneController
func NewSceneController(goveeClient *govee.Client, logger zerolog.Logger) *SceneController {
	return &SceneController{
		activeScenes: make(map[string]context.CancelFunc),
		goveeClient:  goveeClient,
		logger:       logger.With().Str("component", "sceneController").Logger(),
	}
}

// SetScene sets a dynamic scene for a Govee device
func (sc *SceneController) SetScene(goveeLightId string, scene Scene) {
	sc.StopScene(goveeLightId)

	sceneCtx, cancel := context.WithCancel(context.Background())

	sc.mu.Lock()
	sc.activeScenes[goveeLightId] = cancel
	sc.mu.Unlock()

	go sc.runDynamicScene(sceneCtx, goveeLightId, scene)
}

// IsActive returns true if a dynamic scene is currently active for a Govee device
func (sc *SceneController) IsActive(goveeLightID string) bool {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	_, ok := sc.activeScenes[goveeLightID]
	return ok
}

// StopScene stops a dynamic scene for a Govee device
func (sc *SceneController) StopScene(goveeDeviceID string) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	if cancelFunc, exists := sc.activeScenes[goveeDeviceID]; exists {
		cancelFunc()
		delete(sc.activeScenes, goveeDeviceID)
	}
}

// runDynamicScene runs a dynamic scene for a Govee device
func (sc *SceneController) runDynamicScene(ctx context.Context, goveeDeviceID string, scene Scene) {
	if len(scene.Palette.Color) == 0 {
		sc.logger.Warn().Str("deviceId", goveeDeviceID).Msg("Scene has no colors in palette")
		return
	}

	baseCycleTime := 20.0
	adjustedCycleTime := baseCycleTime / scene.Speed
	colorsInPalette := len(scene.Palette.Color)
	timePerColor := time.Duration(adjustedCycleTime/float64(colorsInPalette)) * time.Second
	transitionTime := timePerColor / 3

	sc.logger.Info().
		Float64("sceneSpeed", scene.Speed).
		Dur("timePerColor", timePerColor).
		Dur("transitionTime", transitionTime).
		Int("colorsInPalette", colorsInPalette).
		Msg("Starting dynamic scene")

	for {
		select {
		case <-ctx.Done():
			sc.logger.Info().Str("deviceId", goveeDeviceID).Msg("Stopping dynamic scene")
			return
		default:
			for i, paletteColor := range scene.Palette.Color {
				select {
				case <-ctx.Done():
					return
				default:
					x := paletteColor.Color.XY.X
					y := paletteColor.Color.XY.Y

					brightness := 0.0
					for _, action := range scene.Actions {
						brightness += action.Action.Dimming.Brightness
					}
					brightness /= float64(len(scene.Actions)) // Average brightness across actions

					r, g, b := coordsToRGB(x, y, int(brightness), GamutTypeC, Gamut{})

					sc.logger.Debug().
						Int("paletteIndex", i).
						Float64("x", x).
						Float64("y", y).
						Float64("brightness", brightness).
						Int("r", r).Int("g", g).Int("b", b).
						Msg("Applying palette color")

					if err := sc.goveeClient.SetColor(goveeDeviceID, r, g, b); err != nil {
						sc.logger.Error().Err(err).Str("deviceId", goveeDeviceID).Msg("Failed to set Govee color")
					}

					briByte := int(brightness)
					if err := sc.goveeClient.SetBrightness(goveeDeviceID, briByte); err != nil {
						sc.logger.Error().Err(err).Str("deviceId", goveeDeviceID).Msg("Failed to set Govee brightness")
					}

					select {
					case <-ctx.Done():
						return
					case <-time.After(timePerColor):
					}
				}
			}
		}
	}
}
