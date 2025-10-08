package hue

// hueResponse is a generic response from the Hue API
type hueResponse[T any] struct {
	Errors []struct {
		Type string `json:"type"`
	}
	Data []T `json:"data"`
}

// Coords represents a coordinate pair
type Coords struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// On represents the on/off state of a light
type On struct {
	On bool `json:"on"`
}

// Dimming represents the brightness of a light
type Dimming struct {
	Brightness float64 `json:"brightness"`
}

// ColorTemperature represents the color temperature of a light
type ColorTemperature struct {
	Mirek      int  `json:"mirek"`
	MirekValid bool `json:"mirek_valid"`
}

// Gamut represents the color gamut of a light
type Gamut struct {
	Red   Coords `json:"red"`
	Green Coords `json:"green"`
	Blue  Coords `json:"blue"`
}

// GamutType represents the color gamut type of a light
type GamutType string

const (
	GamutTypeA GamutType = "A"
	GamutTypeB GamutType = "B"
	GamutTypeC GamutType = "C"
)

// Color represents the color of a light
type Color struct {
	XY        Coords    `json:"xy"`
	Gamut     Gamut     `json:"gamut"`
	GamutType GamutType `json:"gamut_type"`
}

// DynamicsStatus represents the status of dynamic color palette
type DynamicsStatus string

const (
	DynamicsStatusActive   DynamicsStatus = "dynamic_palette"
	DynamicsStatusInactive DynamicsStatus = "none"
)

// Dynamics contains the status of dynamic color palette
type Dynamics struct {
	Status DynamicsStatus `json:"status"`
}

// Light represents a Hue light
type Light struct {
	On               On               `json:"on"`
	Dimming          Dimming          `json:"dimming"`
	ColorTemperature ColorTemperature `json:"color_temperature"`
	Color            Color            `json:"color"`
	Dynamics         Dynamics         `json:"dynamics"`
}

// Group represents a Hue group
type Group struct {
	ID   string `json:"rid"`
	Type string `json:"rtype"`
}

// Scene represents a Hue scene
type Scene struct {
	ID      string        `json:"id"`
	Palette Palette       `json:"palette"`
	Speed   float64       `json:"speed"`
	Status  SceneStatus   `json:"status"`
	Group   Group         `json:"group"`
	Actions []SceneAction `json:"actions"`
}

// SceneAction represents a single action in a scene
type SceneAction struct {
	Action struct {
		On      On      `json:"on"`
		Dimming Dimming `json:"dimming"`
	} `json:"action"`
}

// Palette represents a Hue palette
type Palette struct {
	Color            []PaletteColor     `json:"color"`
	Dimming          []interface{}      `json:"dimming"`
	ColorTemperature []PaletteColorTemp `json:"color_temperature"`
}

// PaletteColor represents a single color in a palette
type PaletteColor struct {
	Color struct {
		XY Coords `json:"xy"`
	} `json:"color"`
	Dimming Dimming `json:"dimming"`
}

// PaletteColorTemp represents a single color temperature in a palette
type PaletteColorTemp struct {
	ColorTemperature ColorTemperature `json:"color_temperature"`
	Dimming          Dimming          `json:"dimming"`
}

// SceneStatus represents the status of a scene
type SceneStatus struct {
	Active     string `json:"active"`
	LastRecall string `json:"last_recall,omitempty"`
}

// DiscoveryResponse represents the response from the Hue bridge discovery endpoint
type DiscoveryResponse struct {
	Address string
}
