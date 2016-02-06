package pocketsv

import (
	log "github.com/Sirupsen/logrus"
	hv "github.com/SpectoLabs/hoverfly"
	"net/http"

	"fmt"
)

func Start(mode, databasePath string) {
	log.SetFormatter(&log.TextFormatter{})

	log.Info("--------Hoverfly mobile is starting!!-------")

	log.SetFormatter(&log.JSONFormatter{})

	cfg := hv.InitSettings()

	// overriding default settings
	cfg.Mode = mode

	// overriding destination
	cfg.Destination = "."

	//cfg.DatabaseName = "/data/data/dev.client.android/databases/clientDB.db"
	cfg.DatabaseName = fmt.Sprintf("%s/requests.db", databasePath)

	proxy, dbClient := hv.GetNewHoverfly(cfg)
	defer dbClient.Cache.DS.Close()

	// starting admin interface
	dbClient.StartAdminInterface()

	log.Info("Admin interface started")
	// start metrics registry flush

	dbClient.Counter.Init()

	log.Info("Metrics counter started")

	log.Info("Starting main proxy...")
	log.Warn(http.ListenAndServe(fmt.Sprintf(":%s", cfg.ProxyPort), proxy))

}

func Greet(mode string) string {
	return fmt.Sprintf("Hoverfly ready, %s mode!", mode)
}
