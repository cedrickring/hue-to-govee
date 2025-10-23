package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// Synchronization represents a single synchronization config between a Hue light and a Govee device.
type Synchronization struct {
	HueLightId      string `mapstructure:"hue_light_id"`
	HueRoomId       string `mapstructure:"hue_room_id"`
	GoveeDeviceId   string `mapstructure:"govee_device_id"`
	FixedBrightness *int   `mapstructure:"fixed_brightness"`
}

// MustLoad loads the config file and panics if it fails.
func MustLoad() {
	viper.SetConfigFile("config.yaml")
	if err := viper.ReadInConfig(); err != nil {
		panic("Failed to read config file: " + err.Error())
	}
}

// GetSynchronizations returns the synchronizations section of the config.
func GetSynchronizations() ([]Synchronization, error) {
	var synchronizations []Synchronization
	if err := viper.UnmarshalKey("synchronizations", &synchronizations); err != nil {
		return nil, err
	}

	for _, synchronization := range synchronizations {
		if synchronization.FixedBrightness != nil {
			if *synchronization.FixedBrightness > 100 || *synchronization.FixedBrightness < 0 {
				return nil, fmt.Errorf("fixed brightness out of range, must be between 0 and 100")
			}
		}
	}
	return synchronizations, nil
}
