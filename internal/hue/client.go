package hue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/cedrickring/hue-to-govee/internal/logger"
	"github.com/hashicorp/mdns"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Client is a client for the Hue V2 API
type Client struct {
	hueBridgeID string
	logger      zerolog.Logger

	lock          sync.Mutex // Mutex to protect bridgeAddress updates
	bridgeAddress string

	httpClient *http.Client
}

// NewClient creates a new Client with the given hueBridgeID and hueUsername.
func NewClient(hueBridgeID, hueUsername string, logger zerolog.Logger) *Client {
	client := &http.Client{
		Transport: newHueTransport(hueUsername),
	}

	return &Client{
		httpClient:  client,
		hueBridgeID: hueBridgeID,
		logger:      logger,
	}
}

// StartAutoDiscovery starts the auto discovery process to find the Hue bridge.
func (c *Client) StartAutoDiscovery(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	bridge, err := c.discoverBridge(ctx)
	if err != nil {
		return fmt.Errorf("failed to discover Hue bridges: %w", err)
	}

	c.lock.Lock()
	defer c.lock.Unlock()
	c.bridgeAddress = bridge.Address
	c.logger.Info().Str("bridgeID", c.hueBridgeID).Str("address", c.bridgeAddress).Msg("Found Hue bridge")

	go func() {
		for {
			select {
			case <-ctx.Done():
				return

			case <-time.After(10 * time.Second):
				ctx, cancel := context.WithTimeout(ctx, 10*time.Second)

				bridge, err := c.discoverBridge(ctx)
				cancel()
				if err != nil {
					if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
						c.logger.Info().Msg("Discovery canceled or timed out")
						return
					}

					c.logger.Error().Err(err).Msg("Failed to discover Hue bridges")
					continue
				}

				if bridge.Address != "" && bridge.Address != c.bridgeAddress {
					c.logger.Info().Str("bridgeID", c.hueBridgeID).Str("newAddress", bridge.Address).
						Msg("Hue bridge address changed, updating")
					c.lock.Lock()
					c.bridgeAddress = bridge.Address
					c.lock.Unlock()
				} else {
					c.logger.Info().Str("bridgeID", c.hueBridgeID).Msg("No change in Hue bridge address")
				}
			}
		}
	}()

	return nil
}

// GetLight returns the light with the given ID.
func (c *Client) GetLight(lightID string) (*Light, error) {
	defer c.lock.Unlock()
	c.lock.Lock()
	url := fmt.Sprintf("https://%s/clip/v2/resource/light/%s", c.bridgeAddress, lightID)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get light info: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var hueResp hueResponse[Light]
	err = json.Unmarshal(body, &hueResp)
	if err != nil {
		return nil, err
	}

	if len(hueResp.Data) == 0 {
		return nil, fmt.Errorf("no light found with ID %s", lightID)
	}

	return &hueResp.Data[0], nil
}

// GetActiveScene returns the active scene for the room with the given ID.
func (c *Client) GetActiveScene(roomId string) (*Scene, error) {
	defer c.lock.Unlock()
	c.lock.Lock()
	url := fmt.Sprintf("https://%s/clip/v2/resource/scene", c.bridgeAddress)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get active scene: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var hueResp hueResponse[Scene]
	err = json.Unmarshal(body, &hueResp)
	if err != nil {
		return nil, err
	}

	if len(hueResp.Data) == 0 {
		return nil, fmt.Errorf("no active scene found for room ID %s", roomId)
	}

	// Filter scenes by room ID
	for _, scene := range hueResp.Data {
		if scene.Group.ID == roomId && scene.Status.Active == "dynamic_palette" && scene.Group.Type == "room" {
			return &scene, nil
		}
	}

	return nil, fmt.Errorf("no active scene found for room ID %s", roomId)
}

// discoverBridge discovers the Hue bridge using mDNS.
func (c *Client) discoverBridge(ctx context.Context) (*DiscoveryResponse, error) {
	entriesCh := make(chan *mdns.ServiceEntry, 1)

	params := mdns.DefaultParams("_hue._tcp")
	params.Entries = entriesCh
	params.DisableIPv6 = true // Disable IPv6 to avoid issues with some networks
	params.Timeout = 10 * time.Second
	params.Logger = logger.Discard()

	go func() {
		defer close(entriesCh)
		if err := mdns.QueryContext(ctx, params); err != nil {
			log.Error().Err(err).Msg("mDNS query failed")
		}
	}()

	// the service name consists of "Hue Bridge - " followed by the last 6 characters of the hueBridgeID
	serviceName := fmt.Sprintf("Hue Bridge - %s", strings.ToUpper(c.hueBridgeID[len(c.hueBridgeID)-6:]))
	log.Debug().Str("serviceName", serviceName).Msg("Starting mDNS query for service")

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case entry, ok := <-entriesCh:
			if !ok {
				return nil, fmt.Errorf("service '%s' not found", serviceName)
			}

			entryName := strings.ReplaceAll(entry.Name, "\\", "")
			if entryName == "" {
				return nil, fmt.Errorf("service '%s' not found", serviceName)
			}
			entryName = entryName[:len(entryName)-len("._hue._tcp.local.")]
			if strings.EqualFold(entryName, serviceName) {
				serviceInfo := &DiscoveryResponse{
					Address: entry.AddrV4.String(),
				}
				return serviceInfo, nil
			}
		}
	}
}
