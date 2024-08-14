package pool

import (
	"github.com/opensourceways/xihe-extra-services/async-server/domain/repository"
	"github.com/sirupsen/logrus"
)

type TaskList []func()

type Pool interface {
	GetIdleWorker() int
	DoTasks(TaskList) error
}

func (r *TaskList) InitTaskList(reqs []repository.WuKongTask, f func(*repository.WuKongTask) error) {
	*r = make(TaskList, len(reqs))

	// build new function with new address
	funcBuild := func(i int) func() {
		return func() {
			if err := f(&reqs[i]); err != nil {
				logrus.Errorf("Error processing task %d: %v", i, err)
			}
		}
	}

	for i := range reqs {
		([]func())(*r)[i] = funcBuild(i)
	}
}

func (r *TaskList) InitTaskListForWuKong4Img(reqs []repository.WuKongTask, f func(*repository.WuKongTask) error) {
	*r = make(TaskList, len(reqs))

	// build new function with new address
	funcBuild := func(i int) func() {
		return func() {
			if err := f(&reqs[i]); err != nil {
				logrus.Errorf("Error processing task %d: %v", i, err)
			}
		}
	}

	for i := range reqs {
		([]func())(*r)[i] = funcBuild(i)
	}
}
