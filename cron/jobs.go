package cron

import (
	"fmt"
	cron "github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
	"k8s.io/mouse/controller"
	"k8s.io/mouse/model"
	"sync"
)

var Jobs = make(map[string]*Job)

type Job struct {
	sync.Mutex
	CronScale model.CronScaleV1
	Cron      *cron.Cron
}

func (j Job) Run() {
	controller.ReconHub.Add(j.CronScale)
}

func (j *Job) UpdateCronScale(cs model.CronScaleV1) {
	j.Lock()
	j.CronScale = cs
	j.Unlock()
}

func MatchJobs(all []model.CronScaleV1) {
	logrus.Debugln(fmt.Sprintf("> Matching %v jobs", len(all)))
	//Find invalid jobs
	toRemove := make([]string, 0)
	for csID, _ := range Jobs {
		if exists, cs := IDExists(csID, all); !exists || cs.Spec.CronSpec != Jobs[csID].CronScale.Spec.CronSpec {
			logrus.Infoln("... should remove", csID, "for update:", exists)
			toRemove = append(toRemove, csID)
			continue
		}
		logrus.Debugln("... no problem with", csID)
		continue
	}
	//Remove invalid jobs
	for _, tr := range toRemove {
		logrus.Infoln(fmt.Sprintf("> Removed job %s", tr))
		Jobs[tr].Cron.Stop()
		delete(Jobs, tr)
	}

	//Create if not exists
	for _, cs := range all {
		foundCS, found := Jobs[cs.GetID()]
		if found {
			logrus.Debugln("...", cs.GetID(), "updating as already exists")
			foundCS.UpdateCronScale(cs)
			continue
		}
		logrus.Infoln("...", cs.GetID(), "should be created")
		c := cron.New()
		j := &Job{
			CronScale: cs,
			Cron:      c,
		}
		_, err := c.AddJob(cs.Spec.CronSpec, j)
		c.Start()
		if err != nil {
			logrus.Errorln(err)
			continue
		}
		logrus.Infoln(fmt.Sprintf("> Creating job %s as:", cs.GetID()))
		logrus.Infoln(cs.PrettyString())
		Jobs[cs.GetID()] = j
	}

}

func IDExists(id string, list []model.CronScaleV1) (ok bool, cs model.CronScaleV1) {
	for _, cs = range list {
		if cs.GetID() == id {
			ok = true
			return
		}
	}
	return
}
