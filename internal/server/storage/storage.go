package storage

import (
	"context"

	"github.com/gopherlearning/track-devops/internal"
	"github.com/gopherlearning/track-devops/internal/repositories"
	"github.com/gopherlearning/track-devops/internal/server/storage/local"
	"github.com/gopherlearning/track-devops/internal/server/storage/postgres"
	"github.com/sirupsen/logrus"
)

func InitStorage(args internal.ServerArgs) (store repositories.Repository, err error) {
	if len(args.DatabaseDSN) != 0 {
		store, err = postgres.NewStorage(args.DatabaseDSN, logrus.StandardLogger())
		if err != nil {
			return nil, err
		}
	} else {
		store, err = local.NewStorage(args.Restore, &args.StoreInterval, args.StoreFile)
		if err != nil {
			return nil, err
		}
	}
	return store, nil
}

func CloseStorage(args internal.ServerArgs, store repositories.Repository) (err error) {
	if len(args.DatabaseDSN) != 0 {
		err = store.(*postgres.Storage).Close(context.Background())
		if err != nil {
			return err
		}
	} else {
		err = store.(*local.Storage).Save()
		if err != nil {
			return err
		}
	}
	return nil
}
