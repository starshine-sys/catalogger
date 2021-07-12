package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/ReneKroon/ttlcache/v2"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/mediocregopher/radix/v4"
	"google.golang.org/grpc"

	_ "github.com/joho/godotenv/autoload"
	"github.com/julienschmidt/httprouter"
	basedb "github.com/starshine-sys/catalogger/db"
	"github.com/starshine-sys/catalogger/web/frontend/db"
	"github.com/starshine-sys/catalogger/web/proto"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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

	debug, _ := strconv.ParseBool(os.Getenv("DEBUG_LOGGING"))

	// set up a logger
	zcfg := zap.NewProductionConfig()
	zcfg.Encoding = "console"
	zcfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	zcfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	zcfg.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
	zcfg.EncoderConfig.EncodeDuration = zapcore.StringDurationEncoder

	if debug {
		zcfg.Level.SetLevel(zapcore.DebugLevel)
	} else {
		zcfg.Level.SetLevel(zapcore.InfoLevel)
	}

	zap, err := zcfg.Build(zap.AddStacktrace(zapcore.ErrorLevel))
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

func (s *server) multiRedis(ctx context.Context, cmds ...radix.Action) error {
	for _, cmd := range cmds {
		err := s.Redis.Do(ctx, cmd)
		if err != nil {
			return err
		}
	}
	return nil
}

type userCache struct {
	*api.Client
	User *discord.User
}

func (s *server) getUser(cookie string) (*userCache, bool) {
	v, err := s.UserCache.Get(cookie[:80])
	if err == nil {
		c, ok := v.(*userCache)
		if ok {
			return c, true
		}
	}
	return nil, false
}

func (s *server) setUser(cookie string, c *userCache) {
	s.UserCache.Set(cookie[:80], c)
}
