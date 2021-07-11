package main

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/ReneKroon/ttlcache/v2"
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
)

type server struct {
	RPC proto.GuildInfoServiceClient

	DB    *db.DB
	Mux   *httprouter.Router
	Sugar *zap.SugaredLogger

	CSRFTokens  *ttlcache.Cache
	ClientCache *ttlcache.Cache
}

func main() {
	if rpcHost == "" {
		rpcHost = "localhost:50051"
	}

	if clientID == "" || clientSecret == "" || databaseURL == "" || rpcHost == "" || port == "" {
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

	database, err := basedb.New(databaseURL, sugar, nil)
	if err != nil {
		sugar.Fatalf("Error connecting to database: %v", err)
	}

	s := &server{
		DB:          &db.DB{DB: database},
		CSRFTokens:  ttlcache.NewCache(),
		ClientCache: ttlcache.NewCache(),
		Sugar:       sugar,
	}
	s.CSRFTokens.SetCacheSizeLimit(1000)
	s.ClientCache.SetCacheSizeLimit(10000)
	s.ClientCache.SetTTL(time.Hour)
	s.Mux = newRouter(s)

	// connect to RPC server
	conn, err := grpc.Dial(rpcHost, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		sugar.Fatalf("Could not connect to RPC server: %v", err)
	}

	defer func() {
		sugar.Infof("Closing database connection...")
		s.DB.Pool.Close()
		sugar.Infof("Closing RPC connection...")
		conn.Close()
	}()

	s.RPC = proto.NewGuildInfoServiceClient(conn)

	sugar.Fatal(http.ListenAndServe(port, s.Mux))
}
