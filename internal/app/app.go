package app

import (
	"context"
	"time"

	"github.com/core-go/health"
	ch "github.com/core-go/health/cassandra"
	"github.com/core-go/log/zap"
	"github.com/gocql/gocql"

	"go-service/internal/user"
)

const (
	Keyspace = `masterdata`

	CreateKeyspace = `create keyspace if not exists masterdata with replication = {'class':'SimpleStrategy', 'replication_factor':1}`

	CreateTable = `
					create table if not exists users (
					id varchar,
					username varchar,
					email varchar,
					phone varchar,
					date_of_birth date,
					primary key (id)
	)`
)

type ApplicationContext struct {
	Health *health.Handler
	User   user.UserTransport
}

func NewApp(ctx context.Context, cfg Config) (*ApplicationContext, error) {
	// connect to the cluster
	cluster := gocql.NewCluster(cfg.Cql.PublicIp)
	cluster.Consistency = gocql.Quorum
	cluster.ProtoVersion = 4
	cluster.Timeout = time.Second * 1000
	cluster.ConnectTimeout = time.Second * 1000
	cluster.Authenticator = gocql.PasswordAuthenticator{Username: cfg.Cql.UserName, Password: cfg.Cql.Password}
	session, err := cluster.CreateSession()
	if err != nil {
		return nil, err
	}
	//defer session.Close()

	// create keyspaces
	err = session.Query(CreateKeyspace).Exec()
	if err != nil {
		return nil, err
	}

	//switch keyspaces
	session.Close()
	cluster.Keyspace = Keyspace
	session, err = cluster.CreateSession()
	if err != nil {
		return nil, err
	}
	defer session.Close()

	// create table
	err = session.Query(CreateTable).Exec()
	if err != nil {
		return nil, err
	}

	logError := log.LogError

	userHandler, err := user.NewUserHandler(cluster, logError)
	if err != nil {
		return nil, err
	}

	cqlChecker := ch.NewHealthChecker(cluster)
	healthHandler := health.NewHandler(cqlChecker)

	return &ApplicationContext{
		Health: healthHandler,
		User:   userHandler,
	}, nil
}
