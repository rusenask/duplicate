/* Copyright (C) Karolis Rusenas - All Rights Reserved
 * Unauthorized copying of this file, via any medium is strictly prohibited
 * Proprietary and confidential
 * Written by Karolis Rusenas <karolis.rusenas@gmail.com>, February 2016
 */

package duplicate

import (
	"net/http"

	log "github.com/Sirupsen/logrus"
	hv "github.com/SpectoLabs/hoverfly"
	ui "github.com/rusenask/duplicate/ui"

	"fmt"
)

func Start(mode, databasePath, ipAddress string) {
	log.SetFormatter(&log.TextFormatter{})

	log.Info("--------Duplicate mobile is starting!!-------")

	log.SetFormatter(&log.JSONFormatter{})

	cfg := hv.InitSettings()

	// setting debug mode for UI development
	// FIXME: change when deploying to the phone
	cfg.Development = false

	// overriding default settings
	cfg.Mode = mode

	// overriding destination
	cfg.Destination = "."

	cfg.DatabaseName = fmt.Sprintf("%s/requests.db", databasePath)

	proxy, dbClient := hv.GetNewHoverfly(cfg)

	mc := &ui.MasterConfiguration{
		HDB:       &dbClient,
		IPAddress: ipAddress,
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
