package connection

import (
	"github.com/CyCoreSystems/ari/v6"
	"github.com/CyCoreSystems/ari/v6/client/native"
	"github.com/wbluan/api-stasis-go/config"
)

func ConnectToAri() (ari.Client, error) {
	conf := config.GetAriConfig()

	cl, err := native.Connect(&native.Options{
		Application:  conf.Application,
		Username:     conf.Username,
		Password:     conf.Password,
		URL:          conf.URL,
		WebsocketURL: conf.WebsocketURL,
	})
	return cl, err
}
