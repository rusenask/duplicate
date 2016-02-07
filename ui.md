package duplicate

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	// static assets
	_ "github.com/SpectoLabs/hoverfly/statik"
	"github.com/rakyll/statik/fs"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/negroni"
	"github.com/go-zoo/bone"
	"github.com/gorilla/websocket"
	"github.com/meatballhat/negroni-logrus"

	hv "github.com/SpectoLabs/hoverfly"
)

// MasterConfiguration - holds new web ui and acts as a container to hold hoverfly configuration
type MasterConfiguration struct {
	hdb *hv.DBClient
}

// recordedRequests struct encapsulates payload data
type recordedRequests struct {
	Data []hv.Payload `json:"data"`
}

type recordsCount struct {
	Count int `json:"count"`
}

type statsResponse struct {
	Stats        hv.Stats `json:"stats"`
	RecordsCount int      `json:"recordsCount"`
}

type stateRequest struct {
	Mode        string `json:"mode"`
	Destination string `json:"destination"`
}

type messageResponse struct {
	Message string `json:"message"`
}

// StartWebUI - UI used by android WebView
func (d *MasterConfiguration) StartWebUI() {
	go func() {
		// starting admin interface
		mux := getBoneRouter(*d)
		n := negroni.Classic()

		logLevel := log.WarnLevel

		n.Use(negronilogrus.NewCustomMiddleware(logLevel, &log.JSONFormatter{}, "admin"))
		n.UseHandler(mux)

		// admin interface starting message
		log.WithFields(log.Fields{
			"AdminPort": d.hdb.Cfg.AdminPort,
		}).Info("Admin interface is starting...")

		n.Run(fmt.Sprintf(":%s", d.hdb.Cfg.AdminPort))
	}()
}

// getBoneRouter returns mux for admin interface
func getBoneRouter(d MasterConfiguration) *bone.Mux {
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

	if d.hdb.Cfg.Development {
		mux.Handle("/*", http.FileServer(http.Dir("static/dist")))
	} else {
		mux.Handle("/*", http.FileServer(statikFS))
	}

	return mux
}

// AllRecordsHandler returns JSON content type http response
func (d *MasterConfiguration) AllRecordsHandler(w http.ResponseWriter, req *http.Request) {
	records, err := d.hdb.Cache.GetAllRequests()

	if err == nil {

		w.Header().Set("Content-Type", "application/json")

		var response recordedRequests
		response.Data = records
		b, err := json.Marshal(response)

		if err != nil {
			log.Error(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		} else {
			w.Write(b)
			return
		}
	} else {
		log.WithFields(log.Fields{
			"Error": err.Error(),
		}).Error("Failed to get data from cache!")

		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(500) // can't process this entity
		return
	}
}

// RecordsCount returns number of captured requests as a JSON payload
func (d *MasterConfiguration) RecordsCount(w http.ResponseWriter, req *http.Request) {
	count, err := d.hdb.Cache.RecordsCount()

	if err == nil {

		w.Header().Set("Content-Type", "application/json")

		var response recordsCount
		response.Count = count
		b, err := json.Marshal(response)

		if err != nil {
			log.Error(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		} else {
			w.Write(b)
			return
		}
	} else {
		log.WithFields(log.Fields{
			"Error": err.Error(),
		}).Error("Failed to get data from cache!")

		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(500) // can't process this entity
		return
	}
}

// StatsHandler - returns current stats about Hoverfly (request counts, record count)
func (d *MasterConfiguration) StatsHandler(w http.ResponseWriter, req *http.Request) {
	stats := d.hdb.Counter.Flush()

	count, err := d.hdb.Cache.RecordsCount()

	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	var sr statsResponse
	sr.Stats = stats
	sr.RecordsCount = count

	w.Header().Set("Content-Type", "application/json")

	b, err := json.Marshal(sr)

	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		w.Write(b)
		return
	}

}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// StatsWSHandler - returns current stats about Hoverfly (request counts, record count) through the websocket
func (d *MasterConfiguration) StatsWSHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			return
		}
		log.WithFields(log.Fields{
			"message": string(p),
		}).Info("Got message...")

		for _ = range time.Tick(1 * time.Second) {

			count, err := d.hdb.Cache.RecordsCount()

			if err != nil {
				log.WithFields(log.Fields{
					"message": p,
					"error":   err.Error(),
				}).Error("got error while trying to get records count")
				return
			}

			stats := d.hdb.Counter.Flush()

			var sr statsResponse
			sr.Stats = stats
			sr.RecordsCount = count

			b, err := json.Marshal(sr)

			if err = conn.WriteMessage(messageType, b); err != nil {
				log.WithFields(log.Fields{
					"message": p,
					"error":   err.Error(),
				}).Error("Got error when writing message...")
				return
			}
		}

	}

}

// ImportRecordsHandler - accepts JSON payload and saves it to cache
func (d *MasterConfiguration) ImportRecordsHandler(w http.ResponseWriter, req *http.Request) {

	var requests recordedRequests

	defer req.Body.Close()
	body, err := ioutil.ReadAll(req.Body)

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	var response messageResponse

	if err != nil {
		// failed to read response body
		log.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("Could not read response body!")
		response.Message = "Bad request. Nothing to import!"
		http.Error(w, "Failed to read request body.", 400)
		return
	}

	err = json.Unmarshal(body, &requests)

	if err != nil {
		w.WriteHeader(422) // can't process this entity
		return
	}

	payloads := requests.Data
	if len(payloads) > 0 {
		for _, pl := range payloads {
			bts, err := pl.Encode()
			if err != nil {
				log.WithFields(log.Fields{
					"error": err.Error(),
				}).Error("Failed to encode payload")
			} else {
				// recalculating request hash and storing it in database
				r := hv.RequestContainer{Details: pl.Request}
				d.hdb.Cache.Set([]byte(r.Hash()), bts)
			}
		}
		response.Message = fmt.Sprintf("%d requests imported successfully", len(payloads))
	} else {
		response.Message = "Bad request. Nothing to import!"
		w.WriteHeader(400)
	}

	b, err := json.Marshal(response)
	w.Write(b)

}

// DeleteAllRecordsHandler - deletes all captured requests
func (d *MasterConfiguration) DeleteAllRecordsHandler(w http.ResponseWriter, req *http.Request) {
	err := d.hdb.Cache.DeleteBucket(d.hdb.Cache.RequestsBucket)

	w.Header().Set("Content-Type", "application/json")

	var response messageResponse
	if err != nil {
		if err.Error() == "bucket not found" {
			response.Message = fmt.Sprintf("No records found")
			w.WriteHeader(200)
		} else {
			response.Message = fmt.Sprintf("Something went wrong: %s", err.Error())
			w.WriteHeader(500)
		}
	} else {
		response.Message = "Proxy cache deleted successfuly"
		w.WriteHeader(200)
	}
	b, err := json.Marshal(response)

	w.Write(b)
	return
}

// CurrentStateHandler returns current state
func (d *MasterConfiguration) CurrentStateHandler(w http.ResponseWriter, req *http.Request) {
	var resp stateRequest
	resp.Mode = d.hdb.Cfg.GetMode()
	resp.Destination = d.hdb.Cfg.Destination

	b, _ := json.Marshal(resp)
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.Write(b)
}

// StateHandler handles current proxy state
func (d *MasterConfiguration) StateHandler(w http.ResponseWriter, r *http.Request) {
	var sr stateRequest

	// this is mainly for testing, since when you create
	if r.Body == nil {
		r.Body = ioutil.NopCloser(bytes.NewBuffer([]byte("")))
	}

	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)

	if err != nil {
		// failed to read response body
		log.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("Could not read response body!")
		http.Error(w, "Failed to read request body.", 400)
		return
	}

	err = json.Unmarshal(body, &sr)

	if err != nil {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(400) // can't process this entity
		return
	}

	availableModes := map[string]bool{
		"virtualize": true,
		"capture":    true,
		"modify":     true,
		"synthesize": true,
	}

	if !availableModes[sr.Mode] {
		log.WithFields(log.Fields{
			"suppliedMode": sr.Mode,
		}).Error("Wrong mode found, can't change state")
		http.Error(w, "Bad mode supplied, available modes: virtualize, capture, modify, synthesize.", 400)
		return
	}

	log.WithFields(log.Fields{
		"newState": sr.Mode,
		"body":     string(body),
	}).Info("Handling state change request!")

	// setting new state
	d.hdb.Cfg.SetMode(sr.Mode)

	var resp stateRequest
	resp.Mode = d.hdb.Cfg.GetMode()
	resp.Destination = d.hdb.Cfg.Destination
	b, _ := json.Marshal(resp)
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.Write(b)

}
