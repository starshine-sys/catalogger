package server

import (
	"log"
	"net"
	"os"
	"strings"
	"time"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/starshine-sys/bcr"
	"github.com/starshine-sys/catalogger/db"
	"github.com/starshine-sys/catalogger/web/proto"
	"google.golang.org/grpc"
)

var _ proto.GuildInfoServiceServer = (*RPCServer)(nil)

// RPCServer is a gRPC server
type RPCServer struct {
	Bot *bcr.Router
	DB  *db.DB

	clearCache  func(discord.GuildID, ...discord.ChannelID)
	memberCount func() int64
	guildPerms  func(discord.GuildID, discord.UserID) (discord.Guild, discord.Permissions, error)

	proto.UnimplementedGuildInfoServiceServer
}

// NewServer creates a new RPCServer, starts it, and returns it
func NewServer(bot *bcr.Router, db *db.DB, clearCacheFunc func(discord.GuildID, ...discord.ChannelID), memberCountFunc func() int64, guildPermFunc func(discord.GuildID, discord.UserID) (discord.Guild, discord.Permissions, error)) *RPCServer {
	s := &RPCServer{
		Bot:         bot,
		DB:          db,
		clearCache:  clearCacheFunc,
		memberCount: memberCountFunc,
		guildPerms:  guildPermFunc,
	}

	port := strings.TrimPrefix(os.Getenv("RPC_PORT"), ":")
	if port == "" {
		port = "50051"
	}

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	rpcs := grpc.NewServer()
	proto.RegisterGuildInfoServiceServer(rpcs, s)

	db.Sugar.Infof("RPC server listening at %v", lis.Addr())

	go func() {
		for {
			err := rpcs.Serve(lis)
			if err != nil {
				s.DB.Sugar.Errorf("Failed to serve RPC: %v", err)
			}
			time.Sleep(30 * time.Second)
		}
	}()

	return s
}
