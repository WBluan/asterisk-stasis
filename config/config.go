package config

type ARIConfig struct {
	Application  string
	Username     string
	Password     string
	URL          string
	WebsocketURL string
}

func GetAriConfig() ARIConfig {
	return ARIConfig{
		Application:  "simple-call",
		Username:     "asterisk",
		Password:     "asterisk",
		URL:          "http://localhost:8088/ari",
		WebsocketURL: "ws://localhost:8088/ari/events",
	}
}
