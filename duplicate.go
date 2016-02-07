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

	log.Info("--------Duplicate mobile is starting!!-------")

	log.SetFormatter(&log.JSONFormatter{})

	cfg := hv.InitSettings()

	// setting debug mode for UI development
	// FIXME: change when deploying to the phone
	cfg.Development = true

	// overriding default settings
	cfg.Mode = mode

	// overriding destination
	cfg.Destination = "."

	cfg.DatabaseName = fmt.Sprintf("%s/requests.db", databasePath)

	proxy, dbClient := hv.GetNewHoverfly(cfg)

	mc := &MasterConfiguration{
		hdb: &dbClient,
	}
	defer dbClient.Cache.DS.Close()

	// starting admin interface
	//dbClient.StartAdminInterface()
	mc.StartWebUI()

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

type MasterConfiguration struct {
	hdb *hv.DBClient
}

// StartAdminInterface - starts admin interface web server
func (mc *MasterConfiguration) StartWebUI() {
	go func() {
		// starting admin interface
		mux := getBoneRouter(mc)
		n := negroni.Classic()

		logLevel := log.WarnLevel

		n.Use(negronilogrus.NewCustomMiddleware(logLevel, &log.JSONFormatter{}, "admin"))
		n.UseHandler(mux)

		// admin interface starting message
		log.WithFields(log.Fields{
			"UIPort": mc.hdb.Cfg.AdminPort,
		}).Info("Web UI is starting...")

		n.Run(fmt.Sprintf(":%s", mc.hdb.Cfg.AdminPort))
	}()
}

// getBoneRouter returns mux for admin interface
func getBoneRouter(m *MasterConfiguration) *bone.Mux {
	mux := bone.New()

	mux.Get("/records", http.HandlerFunc(m.hdb.AllRecordsHandler))
	mux.Delete("/records", http.HandlerFunc(m.hdb.DeleteAllRecordsHandler))
	mux.Post("/records", http.HandlerFunc(m.hdb.ImportRecordsHandler))

	mux.Get("/count", http.HandlerFunc(m.hdb.RecordsCount))
	mux.Get("/stats", http.HandlerFunc(m.hdb.StatsHandler))
	mux.Get("/statsws", http.HandlerFunc(m.hdb.StatsWSHandler))

	mux.Get("/state", http.HandlerFunc(m.hdb.CurrentStateHandler))
	mux.Post("/state", http.HandlerFunc(m.hdb.StateHandler))

	if m.hdb.Cfg.Development {
		log.Warn("Looking for static files in static/dist instead of binary!")
		mux.Handle("/*", http.FileServer(http.Dir("../../static/dist/")))
	} else {
		// preparing static assets for embedded admin
		statikFS, err := fs.New()
		if err != nil {
			log.WithFields(log.Fields{
				"Error": err.Error(),
			}).Error("Failed to load statikFS, admin UI might not work :(")
		}
		mux.Handle("/*", http.FileServer(statikFS))
	}

	return mux
}
