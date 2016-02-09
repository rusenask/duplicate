package score

import (
	"fmt"

	log "github.com/Sirupsen/logrus"
	"github.com/SpectoLabs/hoverfly"
	ui "github.com/rusenask/duplicate/ui"
)

// ScoresHook adds score based on user
type ScoresHook struct {
	MCfg *ui.MasterConfiguration
}

// NewScoresHook - creates hook to start listening for certain actions to add points to score table
func NewScoresHook(mc *ui.MasterConfiguration) (*ScoresHook, error) {
	return &ScoresHook{MCfg: mc}, nil
}

func (hook *ScoresHook) Fire(entry *hoverfly.Entry) (err error) {

	fmt.Printf("Got entry: %s", entry.Message)
	log.WithFields(log.Fields{
		"actionType": entry.ActionType,
	}).Info("hook got message, firing!!!")

	if entry.ActionType == hoverfly.ActionTypeRequestCaptured {
		hook.MCfg.UserDetails.AddPoints(hoverfly.CaptureMode, 1)
	}

	// TODO: add other remaining scores for:
	// virtualize
	// modify
	// virtualize

	return err
}

// ActionTypes - action types that are interesting for this hook
func (hook *ScoresHook) ActionTypes() []hoverfly.ActionType {
	return []hoverfly.ActionType{
		hoverfly.ActionTypeRequestCaptured,
	}
}
