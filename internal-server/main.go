package main

import (
	"flag"
	"os"
	"strconv"

	"github.com/opensourceways/community-robot-lib/logrusutil"
	liboptions "github.com/opensourceways/community-robot-lib/options"
	"github.com/opensourceways/xihe-grpc-protocol/grpc/competition"
	"github.com/opensourceways/xihe-grpc-protocol/grpc/finetune"
	"github.com/opensourceways/xihe-grpc-protocol/grpc/server"
	"github.com/opensourceways/xihe-grpc-protocol/grpc/training"
	"github.com/sirupsen/logrus"

	"github.com/opensourceways/xihe-server/app"
	"github.com/opensourceways/xihe-server/common/infrastructure/pgsql"
	competitionapp "github.com/opensourceways/xihe-server/competition/app"
	competitiondomain "github.com/opensourceways/xihe-server/competition/domain"
	competitionrepo "github.com/opensourceways/xihe-server/competition/infrastructure/repositoryimpl"
	"github.com/opensourceways/xihe-server/domain"
	"github.com/opensourceways/xihe-server/infrastructure/mongodb"
	"github.com/opensourceways/xihe-server/infrastructure/repositories"
)

type options struct {
	service     liboptions.ServiceOptions
	enableDebug bool
}

func (o *options) Validate() error {
	return o.service.Validate()
}

func (o *options) addFlags(fs *flag.FlagSet) {
	o.service.AddFlags(fs)

	fs.BoolVar(
		&o.enableDebug, "enable_debug", false,
		"whether to enable debug model.",
	)
}

func gatherOptions(fs *flag.FlagSet, args ...string) (options, error) {
	var o options

	o.addFlags(fs)

	err := fs.Parse(args)

	return o, err
}

func main() {
	logrusutil.ComponentInit("xihe")
	log := logrus.NewEntry(logrus.StandardLogger())

	o, err := gatherOptions(
		flag.NewFlagSet(os.Args[0], flag.ExitOnError),
		os.Args[1:]...,
	)
	if err != nil {
		logrus.Fatalf("new options failed, err:%s", err.Error())
	}

	if err := o.Validate(); err != nil {
		logrus.Fatalf("Invalid options, err:%s", err.Error())
	}

	if o.enableDebug {
		logrus.SetLevel(logrus.DebugLevel)
		logrus.Debug("debug enabled.")
	}

	// cfg
	cfg := new(configuration)
	if err := loadConfig(o.service.ConfigFile, cfg); err != nil {
		log.Fatalf("load config, err:%s", err.Error())
	}

	if err := os.Remove(o.service.ConfigFile); err != nil {
		logrus.Fatalf("config file delete failed, err:%s", err.Error())
	}

	// mongo
	m := &cfg.Mongodb
	if err := mongodb.Initialize(m.DBConn, m.DBName, m.DBCert); err != nil {
		log.Fatalf("initialize mongodb failed, err:%s", err.Error())
	}

	// postgresql
	if err := pgsql.Init(&cfg.Postgresql.DB); err != nil {
		logrus.Fatalf("init db, err:%s", err.Error())
	}

	if err := os.Remove(cfg.Postgresql.DB.DBCert); err != nil {
		logrus.Fatalf("postgresql dbcert file delete failed, err:%s", err.Error())
	}

	defer mongodb.Close()

	collections := &cfg.Mongodb.Collections

	// training
	train := app.NewTrainingService(
		nil,
		repositories.NewTrainingRepository(
			mongodb.NewTrainingMapper(collections.Training),
		),
		nil, 0,
	)

	// finetune
	finetuneService := app.NewFinetuneInternalService(
		repositories.NewFinetuneRepository(
			mongodb.NewFinetuneMapper(collections.Finetune),
		),
	)

	// competition
	competitionService := competitionapp.NewCompetitionInternalService(
		competitionrepo.NewWorkRepo(
			mongodb.NewCollection(collections.CompetitionWork),
		),
	)

	// cfg
	cfg.initDomainConfig()

	// server
	s := server.NewServer()

	s.RegisterFinetuneServer(finetuneServer{finetuneService})
	s.RegisterTrainingServer(trainingServer{train})
	s.RegisterCompetitionServer(competitionServer{competitionService})

	if err := s.Run(strconv.Itoa(o.service.Port)); err != nil {
		log.Errorf("start server failed, err:%s", err.Error())
	}
}

type trainingServer struct {
	service app.TrainingService
}

func (t trainingServer) SetTrainingInfo(index *training.TrainingIndex, v *training.TrainingInfo) error {
	u, err := domain.NewAccount(index.User)
	if err != nil {
		return nil
	}

	return t.service.UpdateJobDetail(
		&domain.TrainingIndex{
			Project: domain.ResourceIndex{
				Owner: u,
				Id:    index.ProjectId,
			},
			TrainingId: index.Id,
		},
		&app.JobDetail{
			Duration:   v.Duration,
			Status:     v.Status,
			LogPath:    v.LogPath,
			AimPath:    v.AimZipPath,
			OutputPath: v.OutputZipPath,
		},
	)
}

// finetune
type finetuneServer struct {
	service app.FinetuneInternalService
}

func (t finetuneServer) SetFinetuneInfo(index *finetune.FinetuneIndex, v *finetune.FinetuneInfo) error {
	u, err := domain.NewAccount(index.User)
	if err != nil {
		return nil
	}

	return t.service.UpdateJobDetail(
		&domain.FinetuneIndex{
			Owner: u,
			Id:    index.Id,
		},
		&app.FinetuneJobDetail{
			Duration: v.Duration,
			Status:   v.Status,
		},
	)
}

// competition
type competitionServer struct {
	service competitionapp.CompetitionInternalService
}

func (t competitionServer) SetSubmissionInfo(
	cid string, v *competition.SubmissionInfo,
) error {
	phase, err := competitiondomain.NewCompetitionPhase(v.Phase)
	if err != nil {
		return err
	}

	return t.service.UpdateSubmission(
		&competitionapp.CompetitionSubmissionUpdateCmd{
			Index:  competitiondomain.NewWorkIndex(cid, v.PlayerId),
			Phase:  phase,
			Id:     v.Id,
			Status: v.Status,
			Score:  v.Score,
		},
	)
}