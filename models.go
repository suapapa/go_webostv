package webostv

import (
	"fmt"
)

// Application represents a WebOS application.
type Application struct {
	ID    string                 `json:"id"`
	Title string                 `json:"title"`
	Icon  string                 `json:"icon"`
	Data  map[string]interface{} `json:"-"`
}

func (a Application) String() string {
	return fmt.Sprintf("<Application %q (ID: %s)>", a.Title, a.ID)
}

// InputSource represents a TV input source (e.g., HDMI1).
type InputSource struct {
	ID    string                 `json:"id"`
	Label string                 `json:"label"`
	Data  map[string]interface{} `json:"-"`
}

func (i InputSource) String() string {
	return fmt.Sprintf("<InputSource %q (ID: %s)>", i.Label, i.ID)
}

// AudioOutputSource represents where the audio is played.
type AudioOutputSource struct {
	Source string `json:"soundOutput"`
}

func (a AudioOutputSource) String() string {
	return fmt.Sprintf("<AudioOutputSource %q>", a.Source)
}

// VolumeInfo contains volume related information.
type VolumeInfo struct {
	Volume       int    `json:"volume"`
	Mute         bool   `json:"mute"`
	Scenario     string `json:"scenario"`
	SubScenario  string `json:"subScenario"`
	SoundOutput  string `json:"soundOutput"`
	Action       string `json:"action"` // used in subscriptions
	ChangeReason string `json:"changeReason"`
}

// ChannelInfo contains information about a TV channel.
type ChannelInfo struct {
	ChannelID          string `json:"channelId"`
	ChannelName        string `json:"channelName"`
	ChannelNumber      string `json:"channelNumber"`
	ChannelModeName    string `json:"channelModeName"`
	SignalChannelID    string `json:"signalChannelId"`
	MajorNumber        int    `json:"majorNumber"`
	MinorNumber        int    `json:"minorNumber"`
	PhysicalNumber     int    `json:"physicalNumber"`
	SourceIndex        int    `json:"sourceIndex"`
	ChannelTypeId      int    `json:"channelTypeId"`
	ChannelTypeName    string `json:"channelTypeName"`
	BinaryID           int    `json:"binaryId"`
	ProgramID          string `json:"programId"`
	ChannelType        string `json:"channelType"`
	SatelliteName      string `json:"satelliteName"`
	FavoriteGroup      string `json:"favoriteGroup"`
	Skipped            bool   `json:"skipped"`
	Locked             bool   `json:"locked"`
	Scrambled          bool   `json:"scrambled"`
	ServiceID          int    `json:"serviceId"`
	TransportID        int    `json:"transportId"`
	NetworkID          int    `json:"networkId"`
	ChannelMode        string `json:"channelMode"`
	DualChannel        bool   `json:"dualChannel"`
	TunerID            int    `json:"tunerId"`
	ProgramName        string `json:"programName"`
}

// SWInformation contains software information.
type SWInformation struct {
	ProductBaseVersion string `json:"product_base_version"`
	ModelName          string `json:"model_name"`
	SDKVersion         string `json:"sdk_version"`
	FirmwareVersion    string `json:"firmware_version"`
	BoardType          string `json:"board_type"`
	OTAID              string `json:"ota_id"`
}
