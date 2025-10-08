package govee

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/rs/zerolog"
)

// Construct is a generic message structure for Govee commands
type Construct[DataType any] struct {
	Message Message[DataType] `json:"msg"`
}

// Message is a generic message structure for Govee commands
type Message[DataType any] struct {
	Command string   `json:"cmd"`
	Data    DataType `json:"data"`
}

// DiscoveryData is the data structure for Govee discovery messages
type DiscoveryData struct {
	DeviceID string `json:"device"`
	IP       string `json:"ip"`
}

// DiscoveryResponseData is the data structure for Govee discovery responses
type DiscoveryResponseData struct {
	AccountTopic string `json:"account_topic"`
}

// TurnData is the data structure for Govee turn commands
type TurnData struct {
	Value int `json:"value"`
}

// RGBColor is a representation of an RGB color
type RGBColor struct {
	R int `json:"r"`
	G int `json:"g"`
	B int `json:"b"`
}

// ColorData is the data structure for Govee color commands
type ColorData struct {
	Color            RGBColor `json:"color"`
	ColorTemperature int      `json:"colorTemperature"`
}

// BrightnessData is the data structure for Govee brightness commands
type BrightnessData struct {
	Value int `json:"value"`
}

const (
	discoveryPort int = 4001
	responsePort  int = 4002
	controlPort   int = 4003
)

var (
	ErrDeviceNotFound = fmt.Errorf("device not found")
)

// Client is a client for the Govee API
type Client struct {
	multicastIP string
	logger      zerolog.Logger
	devices     map[string]string // map[deviceID]IP
}

// NewClient creates a new Client
func NewClient(logger zerolog.Logger, multicastIP string) *Client {
	return &Client{
		logger:      logger,
		multicastIP: multicastIP,
		devices:     make(map[string]string),
	}
}

// Discover discovers Govee devices on the local network
func (c *Client) Discover(ctx context.Context) error {
	addr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("%s:%d", c.multicastIP, responsePort))
	if err != nil {
		return fmt.Errorf("failed to resolve multicast address: %w", err)
	}

	conn, err := net.ListenUDP("udp4", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on UDP port %d: %w", responsePort, err)
	}

	go func() {
		buf := make([]byte, 2048)
		for {
			select {
			case <-ctx.Done():
				conn.Close()
				return
			default:
				{
					n, _, err := conn.ReadFromUDP(buf)
					if err != nil {
						c.logger.Error().Err(err).Msg("Failed to read from UDP")
						continue
					}
					var msg Construct[DiscoveryData]
					if err := json.Unmarshal(buf[:n], &msg); err != nil {
						c.logger.Error().Err(err).Msg("Failed to unmarshal discovery message")
						continue
					}
					c.logger.Debug().Any("message", msg).Msg("Received discovery message")
					if msg.Message.Command == "scan" {
						if _, ok := c.devices[msg.Message.Data.DeviceID]; !ok {
							c.devices[msg.Message.Data.DeviceID] = msg.Message.Data.IP
							c.logger.Info().Str("deviceId", msg.Message.Data.DeviceID).
								Str("ip", msg.Message.Data.IP).
								Msg("Found Govee device")
						} else {
							c.logger.Debug().Str("deviceId", msg.Message.Data.DeviceID).
								Str("ip", msg.Message.Data.IP).
								Msg("Govee device already known")
						}
					}
				}

			}

		}
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				{
					c.logger.Debug().Msg("Sending discovery request")
					addr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("%s:%d", c.multicastIP, discoveryPort))
					if err != nil {
						c.logger.Error().Err(err).Msg("Failed to resolve multicast address")
						return
					}

					sock, err := net.DialUDP("udp4", nil, addr)
					if err != nil {
						continue
					}

					query := Construct[DiscoveryResponseData]{
						Message: Message[DiscoveryResponseData]{
							Command: "scan",
							Data:    DiscoveryResponseData{AccountTopic: "reserve"},
						},
					}
					b, _ := json.Marshal(query)
					if _, err := sock.Write(b); err != nil {
						sock.Close()
						continue
					}
					sock.Close()
					<-time.After(2 * time.Second) // wait for responses
				}
			}
		}
	}()

	return nil
}

// sendCommand sends a command to a Govee device
func (c *Client) sendCommand(deviceID string, cmd string, data interface{}) error {
	if ip, ok := c.devices[deviceID]; ok {
		payload := Construct[interface{}]{
			Message: Message[interface{}]{
				Command: cmd,
				Data:    data,
			},
		}
		b, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to marshal command: %w", err)
		}

		conn, err := net.Dial("udp", fmt.Sprintf("%s:%d", ip, controlPort))
		if err != nil {
			return fmt.Errorf("failed to connect to device %s: %w", deviceID, err)
		}
		defer conn.Close()

		if _, err := conn.Write(b); err != nil {
			return fmt.Errorf("failed to send command to device %s: %w", deviceID, err)
		}
		return nil
	}

	return ErrDeviceNotFound
}

// TurnOn turns on a Govee device
func (c *Client) TurnOn(deviceID string) error {
	return c.sendCommand(deviceID, "turn", TurnData{Value: 1})
}

// TurnOff turns off a Govee device
func (c *Client) TurnOff(deviceID string) error {
	return c.sendCommand(deviceID, "turn", TurnData{Value: 0})
}

// SetColor sets the color of a Govee device
func (c *Client) SetColor(deviceID string, r, g, b int) error {
	colorData := ColorData{
		Color: RGBColor{
			R: r,
			G: g,
			B: b,
		},
		ColorTemperature: 0, // Assuming no color temperature adjustment
	}
	return c.sendCommand(deviceID, "colorwc", colorData)
}

// SetBrightness sets the brightness of a Govee device
func (c *Client) SetBrightness(deviceID string, value int) error {
	briData := BrightnessData{
		Value: value,
	}
	return c.sendCommand(deviceID, "brightness", briData)
}

// IsDeviceNotFound checks if an error is caused by a device not being found
func IsDeviceNotFound(err error) bool {
	return err != nil && errors.Is(err, ErrDeviceNotFound)
}
