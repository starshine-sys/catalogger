package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/ReneKroon/ttlcache/v2"
	"github.com/mediocregopher/radix/v4"
	"google.golang.org/grpc"

	_ "github.com/joho/godotenv/autoload"
	"github.com/julienschmidt/httprouter"
	basedb "github.com/starshine-sys/catalogger/db"
	"github.com/starshine-sys/catalogger/logsetup"
	"github.com/starshine-sys/catalogger/web/frontend/db"
	"github.com/starshine-sys/catalogger/web/proto"
	"go.uber.org/zap"
)

var (
	clientID     = os.Getenv("CLIENT_ID")
	clientSecret = os.Getenv("CLIENT_SECRET")
	databaseURL  = os.Getenv("DATABASE_URL")
	rpcHost      = os.Getenv("RPC_HOST")
	port         = os.Getenv("PORT")
	redisHost    = os.Getenv("REDIS")
)

type server struct {
	RPC proto.GuildInfoServiceClient

	DB    *db.DB
	Redis radix.Client
	Mux   *httprouter.Router
	Sugar *zap.SugaredLogger

	UserCache *ttlcache.Cache
}

func main() {
	if rpcHost == "" {
		rpcHost = "localhost:50051"
	}

	if clientID == "" || clientSecret == "" || databaseURL == "" || rpcHost == "" || port == "" || redisHost == "" {
		log.Println("One or more required env variables was empty")
		return
	}

	zap, err := logsetup.SetupLogging()
	if err != nil {
		panic(err)
	}
	sugar := zap.Sugar()

	s := &server{
		Sugar:     sugar,
		UserCache: ttlcache.NewCache(),
	}
	s.Mux = newRouter(s)

	s.UserCache.SetTTL(30 * time.Minute)
	s.UserCache.SetCacheSizeLimit(10000)

	database, err := basedb.New(databaseURL, sugar, nil)
	if err != nil {
		sugar.Fatalf("Error connecting to database: %v", err)
	}
	sugar.Infof("Database connected")
	s.DB = &db.DB{DB: database}

	s.Redis, err = (radix.PoolConfig{}).New(context.Background(), "tcp", redisHost)
	if err != nil {
		sugar.Fatalf("Error connecting to Redis: %v", err)
	}
	sugar.Infof("Redis connected")

	// connect to RPC server
	conn, err := grpc.Dial(rpcHost, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		sugar.Fatalf("Could not connect to RPC server: %v", err)
	}

	sugar.Infof("RPC connected")

	defer func() {
		sugar.Infof("Closing database connection...")
		s.DB.Pool.Close()
		sugar.Infof("Closing RPC connection...")
		conn.Close()
	}()

	s.RPC = proto.NewGuildInfoServiceClient(conn)

	sugar.Fatal(http.ListenAndServe(port, s.Mux))
}
