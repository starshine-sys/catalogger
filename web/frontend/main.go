package frontend

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"emperror.dev/errors"

	"github.com/ReneKroon/ttlcache/v2"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/go-chi/chi/v5"
	"github.com/mediocregopher/radix/v4"
	"google.golang.org/grpc"

	"github.com/starshine-sys/catalogger/common"
	"github.com/starshine-sys/catalogger/db"
	"github.com/starshine-sys/catalogger/web/proto"
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
	Mux   chi.Router

	UserCache *ttlcache.Cache

	news          []discord.Message
	newsChannel   discord.ChannelID
	newsClient    *api.Client
	newsFetchTime time.Time
}

func Main() (err error) {
	if rpcHost == "" {
		rpcHost = "localhost:50051"
	}

	if clientID == "" || clientSecret == "" || databaseURL == "" || rpcHost == "" || port == "" || redisHost == "" {
		return errors.New("one or more required env variables was empty")
	}

	s := &server{
		UserCache: ttlcache.NewCache(),
	}

	if os.Getenv("ANNOUNCEMENTS") != "" && os.Getenv("ANNOUNCEMENTS_TOKEN") != "" {
		s.newsClient = api.NewClient("Bot " + os.Getenv("ANNOUNCEMENTS_TOKEN"))
		id, err := discord.ParseSnowflake(os.Getenv("ANNOUNCEMENTS"))
		if err != nil {
			return fmt.Errorf("couldn't parse \"%v\" as a snowflake", os.Getenv("ANNOUNCEMENTS"))
		}
		s.newsChannel = discord.ChannelID(id)

		news, err := s.newsClient.Messages(s.newsChannel, 5)
		if err != nil {
			common.Log.Fatalf("Couldn't fetch news messages: %v", err)
		}
		s.news = news
		common.Log.Infof("Fetched %v news messages", len(s.news))
		s.newsFetchTime = time.Now()
	}

	if err = s.UserCache.SetTTL(30 * time.Minute); err != nil {
		common.Log.Panic(err)
	}

	s.UserCache.SetCacheSizeLimit(10000)

	s.DB, err = db.New(databaseURL, nil)
	if err != nil {
		return errors.Wrap(err, "connecting to database")
	}
	s.DB.Stats.SetMode(true)

	s.Redis, err = (radix.PoolConfig{}).New(context.Background(), "tcp", redisHost)
	if err != nil {
		return errors.Wrap(err, "connecting to Redis")
	}
	common.Log.Infof("Redis connected")

	// connect to RPC server
	conn, err := grpc.Dial(rpcHost, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return errors.Wrap(err, "connecting to RPC server")
	}

	common.Log.Infof("RPC connected")

	defer func() {
		common.Log.Infof("Closing database connection...")
		s.DB.Pool.Close()
		common.Log.Infof("Closing RPC connection...")
		conn.Close()
	}()

	s.RPC = proto.NewGuildInfoServiceClient(conn)

	s.Mux = newRouter(s)

	return http.ListenAndServe(port, s.Mux)
}
