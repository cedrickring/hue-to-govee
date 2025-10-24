package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cedrickring/hue-to-govee/internal/config"
	"github.com/cedrickring/hue-to-govee/internal/govee"
	"github.com/cedrickring/hue-to-govee/internal/hue"
	"github.com/cedrickring/hue-to-govee/internal/logger"
	"github.com/spf13/viper"
)
import "github.com/rs/zerolog"

func main() {
	config.MustLoad()
	log := logger.Default()

	log.Info().Msg("Starting Hue to Govee bridge")

	hueBridgeID := viper.GetString("hue_bridge_id")
	hueUsername := viper.GetString("hue_bridge_username")

	hueClient := hue.NewClient(hueBridgeID, hueUsername, log.With().Str("component", "hue").Logger())

	ctx, cancel := context.WithCancel(context.Background())
	catchCtrlC(cancel)

	if err := hueClient.StartAutoDiscovery(ctx); err != nil {
		log.Error().Err(err).Msg("Failed to start Hue auto-discovery")
		return
	}

	goveeClient := govee.NewClient(log, viper.GetString("govee_multicast_ip"))
	if err := goveeClient.Discover(ctx); err != nil {
		log.Error().Err(err).Msg("Failed to discover Govee devices")
		return
	}
	log.Info().Msg("Discovering Govee devices")

	sceneController := hue.NewSceneController(goveeClient, log)
	if err := startSynchronization(ctx, log, hueClient, goveeClient, sceneController); err != nil {
		return
	}

	<-ctx.Done()

	log.Info().Msg("Shutting down Hue to Govee bridge")
}

func startSynchronization(ctx context.Context, logger zerolog.Logger, hueClient *hue.Client, goveeClient *govee.Client, sc *hue.SceneController) error {
	synchronizations, err := config.GetSynchronizations()
	if err != nil {
		logger.Error().Err(err).Msg("Failed to load synchronizations from config")
		return fmt.Errorf("failed to load synchronizations: %w", err)
	}

	for _, sync := range synchronizations {
		syncCopy := sync
		logger.Info().Msgf("Synchronizing Hue light %s <--> Govee device %s", syncCopy.HueLightId,
			syncCopy.GoveeDeviceId)

		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case <-time.After(500 * time.Millisecond):
					light, err := hueClient.GetLight(syncCopy.HueLightId)
					if err != nil {
						logger.Error().Err(err).Str("lightId", syncCopy.HueLightId).Msg("Failed to get Hue light")
						continue
					}
					if light.On.On {
						if light.Dynamics.Status == hue.DynamicsStatusActive {
							if sc.IsActive(syncCopy.GoveeDeviceId) {
								logger.Debug().Str("deviceId", syncCopy.GoveeDeviceId).
									Msg("Skipping Govee sync due to active scene")
								continue
							}

							scene, err := hueClient.GetActiveScene(syncCopy.HueRoomId)
							if err != nil {
								logger.Error().Err(err).Str("roomId", syncCopy.HueRoomId).
									Msg("Failed to get active scene for Hue room")
								continue
							}

							if scene == nil {
								logger.Warn().Str("roomId", syncCopy.HueRoomId).
									Msg("No active scene found for Hue room")
								continue
							}

							sc.SetScene(syncCopy.GoveeDeviceId, *scene)
							continue
						} else {
							if sc.IsActive(syncCopy.GoveeDeviceId) {
								sc.StopScene(syncCopy.GoveeDeviceId)
								logger.Info().Str("goveeDeviceId", syncCopy.GoveeDeviceId).
									Msgf("Stopped dynamic scene for Govee device %s", syncCopy.GoveeDeviceId)
							}
						}

						r, g, b := hue.ColorToRGB(light, sync.FixedBrightness)
						bri := int(float64(light.Dimming.Brightness) / 254.0 * 100)
						if sync.FixedBrightness != nil {
							bri = *sync.FixedBrightness
						}

						if err := goveeClient.SetColor(syncCopy.GoveeDeviceId, r, g, b); err != nil {
							if govee.IsDeviceNotFound(err) {
								continue
							}
							logger.Error().Err(err).Str("deviceId",
								syncCopy.GoveeDeviceId).Msg("Failed to set Govee color")
						}
						if err := goveeClient.SetBrightness(syncCopy.GoveeDeviceId, bri); err != nil {
							if govee.IsDeviceNotFound(err) {
								continue
							}

							logger.Error().Err(err).Str("deviceId",
								syncCopy.GoveeDeviceId).Msg("Failed to set Govee brightness")
						}
					} else {
						if err := goveeClient.TurnOff(syncCopy.GoveeDeviceId); err != nil {
							if govee.IsDeviceNotFound(err) {
								continue
							}
							logger.Error().Err(err).Str("deviceId",
								syncCopy.GoveeDeviceId).Msg("Failed to turn off Govee device")
						}
					}
				}
			}
		}()
	}

	return nil
}

// catchCtrlC catches Ctrl+C to gracefully shutdown
func catchCtrlC(cancel context.CancelFunc) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	go func() {
		<-c
		cancel()
	}()
}
