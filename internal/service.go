package internal

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	"testTask/pkg/repository"
)

type Storage interface {
	Registration(ctx context.Context, user repository.User) (int, error)
	Login(ctx context.Context, user repository.User) (repository.User, error)
}

type App struct {
	log   *logrus.Entry
	store Storage
}

func NewApp(log *logrus.Logger, store Storage) *App {
	return &App{
		log:   log.WithField("component", "service"),
		store: store,
	}
}

func (s *App) Registration(ctx context.Context, user repository.User) (int, error) {
	id, err := s.store.Registration(ctx, user)
	if err != nil {
		return 0, fmt.Errorf("err while registration: %w", err)
	}
	return id, nil
}

func (s *App) Login(ctx context.Context, user repository.User) (repository.User, error) {
	user, err := s.store.Login(ctx, user)
	if err != nil {
		return repository.User{}, fmt.Errorf("err while login: %w", err)
	}
	return user, nil
}
