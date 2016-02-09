package ui

import (
	"net/http"

	log "github.com/Sirupsen/logrus"
	hv "github.com/SpectoLabs/hoverfly"

	"github.com/codegangsta/negroni"
	"github.com/go-zoo/bone"
	"github.com/meatballhat/negroni-logrus"

	// static assets
	"github.com/rakyll/statik/fs"
	_ "github.com/rusenask/duplicate/statik"

	"encoding/json"
	"fmt"
	"sync"
)

// Score - container to keep user score
type Score map[string]int

// UserDetails - container for user specific info
type UserDetails struct {
	Username  string `json:"username"`
	UserScore Score  `json:"score"`
	mu        sync.Mutex
}

// TotalScoreName stores total user score
const TotalScoreName = "total"

// AddPoints - add points used to increase points for specific types
// like user.AddPoints("virtualize", 1)
func (ud *UserDetails) AddPoints(name string, points int) {
	ud.mu.Lock()
	ud.UserScore[name] += points
	ud.UserScore[TotalScoreName] += points
	ud.mu.Unlock()
}

// GetScore - used to safely get current user score
func (ud *UserDetails) GetScore() (score Score) {
	ud.mu.Lock()
	score = ud.UserScore
	ud.mu.Unlock()
	return
}

// MasterConfiguration - master configuration
type MasterConfiguration struct {
	HDB         *hv.DBClient
	IPAddress   string
	UserDetails *UserDetails
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
			"UIPort": mc.HDB.Cfg.AdminPort,
		}).Info("Web UI is starting...")

		n.Run(fmt.Sprintf(":%s", mc.HDB.Cfg.AdminPort))
	}()
}

// getBoneRouter returns mux for admin interface
func getBoneRouter(m *MasterConfiguration) *bone.Mux {
	mux := bone.New()

	mux.Get("/records", http.HandlerFunc(m.HDB.AllRecordsHandler))
	mux.Delete("/records", http.HandlerFunc(m.HDB.DeleteAllRecordsHandler))
	mux.Post("/records", http.HandlerFunc(m.HDB.ImportRecordsHandler))

	mux.Get("/count", http.HandlerFunc(m.HDB.RecordsCount))
	mux.Get("/stats", http.HandlerFunc(m.HDB.StatsHandler))
	mux.Get("/statsws", http.HandlerFunc(m.HDB.StatsWSHandler))

	mux.Get("/state", http.HandlerFunc(m.HDB.CurrentStateHandler))
	mux.Post("/state", http.HandlerFunc(m.HDB.StateHandler))

	// duplicate specific details
	mux.Get("/system", http.HandlerFunc(m.GetSystemInformationHandler))
	mux.Get("/user", http.HandlerFunc(m.GetUserScore))

	if m.HDB.Cfg.Development {
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

// SystemInfo - container to hold system information, for now just IP address
type SystemInfo struct {
	IPAddress string `json:"ip"`
}

func (s *SystemInfo) encode() ([]byte, error) {
	return json.Marshal(s)
}

// GetSystemInformationHandler - returns system information, any information about android can go here
func (mc *MasterConfiguration) GetSystemInformationHandler(w http.ResponseWriter, req *http.Request) {
	var si SystemInfo
	si.IPAddress = mc.IPAddress

	w.Header().Set("Content-Type", "application/json")
	bts, err := si.encode()

	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		w.Write(bts)
		return
	}
}

// GetUserScore - gets score for the current user
func (mc *MasterConfiguration) GetUserScore(w http.ResponseWriter, req *http.Request) {
	bts, err := json.Marshal(mc.UserDetails)
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		w.Write(bts)
		return
	}
}
