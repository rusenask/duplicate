/* Copyright (C) Karolis Rusenas - All Rights Reserved
 * Unauthorized copying of this file, via any medium is strictly prohibited
 * Proprietary and confidential
 * Written by Karolis Rusenas <karolis.rusenas@gmail.com>, February 2016
 */

package duplicate

import (
	log "github.com/Sirupsen/logrus"
	hv "github.com/SpectoLabs/hoverfly"

	"github.com/codegangsta/negroni"
	"github.com/go-zoo/bone"
	"github.com/meatballhat/negroni-logrus"

	// static assets
	"github.com/rakyll/statik/fs"
	_ "github.com/rusenask/duplicate/statik"

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

	//mc := &MasterConfiguration{
	//	hdb: &dbClient,
	//}
	defer dbClient.Cache.DS.Close()

	// starting admin interface
	//dbClient.StartAdminInterface()
	StartWebUI(&dbClient)

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

// StartAdminInterface - starts admin interface web server
func StartWebUI(d *hv.DBClient) {
	go func() {
		// starting admin interface
		mux := getBoneRouter(d)
		n := negroni.Classic()

		logLevel := log.WarnLevel

		if d.Cfg.Verbose {
			logLevel = log.DebugLevel
		}

		n.Use(negronilogrus.NewCustomMiddleware(logLevel, &log.JSONFormatter{}, "admin"))
		n.UseHandler(mux)

		// admin interface starting message
		log.WithFields(log.Fields{
			"UIPort": d.Cfg.AdminPort,
		}).Info("Web UI is starting...")

		n.Run(fmt.Sprintf(":%s", d.Cfg.AdminPort))
	}()
}

// getBoneRouter returns mux for admin interface
func getBoneRouter(d *hv.DBClient) *bone.Mux {
	mux := bone.New()

	// preparing static assets for embedded admin
	statikFS, err := fs.New()
	if err != nil {
		log.WithFields(log.Fields{
			"Error": err.Error(),
		}).Error("Failed to load statikFS, admin UI might not work :(")
	}

	mux.Get("/records", http.HandlerFunc(d.AllRecordsHandler))
	mux.Delete("/records", http.HandlerFunc(d.DeleteAllRecordsHandler))
	mux.Post("/records", http.HandlerFunc(d.ImportRecordsHandler))

	mux.Get("/count", http.HandlerFunc(d.RecordsCount))
	mux.Get("/stats", http.HandlerFunc(d.StatsHandler))
	mux.Get("/statsws", http.HandlerFunc(d.StatsWSHandler))

	mux.Get("/state", http.HandlerFunc(d.CurrentStateHandler))
	mux.Post("/state", http.HandlerFunc(d.StateHandler))

	if d.Cfg.Development {
		mux.Handle("/*", http.FileServer(http.Dir("static/dist")))
	} else {
		mux.Handle("/*", http.FileServer(statikFS))
	}

	return mux
}
