package config

import "github.com/spf13/viper"

// Synchronization represents a single synchronization config between a Hue light and a Govee device.
type Synchronization struct {
	HueLightId    string `mapstructure:"hue_light_id"`
	HueRoomId     string `mapstructure:"hue_room_id"`
	GoveeDeviceId string `mapstructure:"govee_device_id"`
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
	return synchronizations, nil
}
