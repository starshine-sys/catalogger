package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/ReneKroon/ttlcache/v2"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/go-chi/chi/v5"
	"github.com/mediocregopher/radix/v4"
	"google.golang.org/grpc"

	_ "github.com/joho/godotenv/autoload"
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

func main() {
	var err error

	if rpcHost == "" {
		rpcHost = "localhost:50051"
	}

	if clientID == "" || clientSecret == "" || databaseURL == "" || rpcHost == "" || port == "" || redisHost == "" {
		log.Println("One or more required env variables was empty")
		return
	}

	s := &server{
		UserCache: ttlcache.NewCache(),
	}
	s.Mux = newRouter(s)

	if os.Getenv("ANNOUNCEMENTS") != "" && os.Getenv("TOKEN") != "" {
		s.newsClient = api.NewClient("Bot " + os.Getenv("TOKEN"))
		id, err := discord.ParseSnowflake(os.Getenv("ANNOUNCEMENTS"))
		if err != nil {
			common.Log.Fatalf("Couldn't parse \"%v\" as a snowflake!", os.Getenv("ANNOUNCEMENTS"))
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
		common.Log.Fatalf("Error connecting to database: %v", err)
	}

	s.Redis, err = (radix.PoolConfig{}).New(context.Background(), "tcp", redisHost)
	if err != nil {
		common.Log.Fatalf("Error connecting to Redis: %v", err)
	}
	common.Log.Infof("Redis connected")

	// connect to RPC server
	conn, err := grpc.Dial(rpcHost, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		common.Log.Fatalf("Could not connect to RPC server: %v", err)
	}

	common.Log.Infof("RPC connected")

	defer func() {
		common.Log.Infof("Closing database connection...")
		s.DB.Pool.Close()
		common.Log.Infof("Closing RPC connection...")
		conn.Close()
	}()

	s.RPC = proto.NewGuildInfoServiceClient(conn)

	common.Log.Fatal(http.ListenAndServe(port, s.Mux))
}
