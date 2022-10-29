package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v4/stdlib"
	migrate "github.com/rubenv/sql-migrate"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"testTask/internal"
	"testTask/internal/rest"
	"testTask/pkg/repository"
)

const pgDSN = "postgres://postgres:secret@localhost:5433/postgres"

type IntegrationTestSuite struct {
	suite.Suite
	log    *logrus.Logger
	store  *repository.PG
	router *rest.Router
	app    *internal.App
	url    string
}

func (s *IntegrationTestSuite) SetupSuite() {
	ctx := context.Background()
	s.log = logrus.New()
	var err error
	s.store, err = repository.NewRepo(ctx, s.log, pgDSN)
	require.NoError(s.T(), err)
	err = s.store.Migrate(migrate.Up)
	require.NoError(s.T(), err)
	s.app = internal.NewApp(s.log, s.store)
	s.router = rest.NewRouter(s.log, s.app)
	go func() {
		_ = s.router.Run(ctx, "localhost:4001")
	}()
	s.url = "http://localhost:4001/api/v1"
	time.Sleep(100 * time.Millisecond)
}

func (s *IntegrationTestSuite) TearDownSuite() {
	err := s.store.Migrate(migrate.Down)
	require.NoError(s.T(), err)
}

func (s *IntegrationTestSuite) TestRegistrationAndLogin() {
	ctx := context.Background()
	user := repository.User{
		Username: "aspan",
		Password: "qwerty987654",
	}
	path := s.url + "/registration"
	resp := s.processRequest(ctx, http.MethodPost, path, user, nil)
	require.Equal(s.T(), http.StatusCreated, resp.StatusCode)

	var userResp repository.User
	path = s.url + "/login"
	resp = s.processRequest(ctx, http.MethodPost, path, user, &userResp)
	require.Equal(s.T(), http.StatusOK, resp.StatusCode)
	require.Equal(s.T(), userResp.Username, user.Username)
}

func (s *IntegrationTestSuite) TestRegisterNonConflict() {
	ctx := context.Background()
	user := repository.User{
		Username: "ans@23",
		Password: "sjdgkjdsk82",
	}
	path := s.url + "/registration"
	resp := s.processRequest(ctx, http.MethodPost, path, user, nil)
	require.Equal(s.T(), http.StatusCreated, resp.StatusCode)

	user = repository.User{
		Username: "sdds@23",
		Password: "sjdgkjdsk82",
	}
	resp = s.processRequest(ctx, http.MethodPost, path, user, nil)
	require.Equal(s.T(), http.StatusCreated, resp.StatusCode)
}

func (s *IntegrationTestSuite) TestRegisterConflict() {
	ctx := context.Background()
	user := repository.User{
		Username: "anuar@23",
		Password: "sjdgkjdsk82",
	}
	path := s.url + "/registration"
	resp := s.processRequest(ctx, http.MethodPost, path, user, nil)
	require.Equal(s.T(), http.StatusCreated, resp.StatusCode)

	user = repository.User{
		Username: "anuar@23",
		Password: "sjdgkjdsk82",
	}
	resp = s.processRequest(ctx, http.MethodPost, path, user, nil)
	require.Equal(s.T(), http.StatusConflict, resp.StatusCode)
}

func (s *IntegrationTestSuite) TestLoginForbidden() {
	ctx := context.Background()
	user := repository.User{
		Username: "Moon@23",
		Password: "sjdgkjdsk82",
	}
	path := s.url + "/registration"
	resp := s.processRequest(ctx, http.MethodPost, path, user, nil)
	require.Equal(s.T(), http.StatusCreated, resp.StatusCode)

	user = repository.User{
		Username: "Moon@23",
		Password: "opopola82",
	}
	path = s.url + "/login"
	resp = s.processRequest(ctx, http.MethodPost, path, user, nil)
	require.Equal(s.T(), http.StatusForbidden, resp.StatusCode)
}

func (s *IntegrationTestSuite) processRequest(ctx context.Context, method, path string, body interface{}, response interface{}) *http.Response {
	s.T().Helper()
	requestBody, err := json.Marshal(body)
	require.NoError(s.T(), err)
	req, err := http.NewRequestWithContext(ctx, method, path, bytes.NewBuffer(requestBody))
	require.NoError(s.T(), err)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(s.T(), err)
	defer func() {
		require.NoError(s.T(), resp.Body.Close())
	}()
	if response != nil {
		err = json.NewDecoder(resp.Body).Decode(response)
		require.NoError(s.T(), err)
	}
	return resp
}

func TestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}
