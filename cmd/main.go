package main

import (
	"context"
	"github.com/Tomas-vilte/GoMusicBot/internal/config"
	"github.com/Tomas-vilte/GoMusicBot/internal/discord"
	"github.com/Tomas-vilte/GoMusicBot/internal/music/fetcher"
	"github.com/bwmarrin/discordgo"
	"github.com/kelseyhightower/envconfig"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"os/signal"
	"syscall"
)

var (
	logger         *zap.Logger
	ctx            context.Context
	cancelCtx      context.CancelFunc
	cfg            = &config.Config{}
	youtubeFetcher *fetcher.YoutubeFetcher
	storage        *discord.InMemoryInteractionStorage
)

func main() {
	loggerCfg := zap.NewDevelopmentConfig()
	loggerCfg = zap.NewDevelopmentConfig()
	loggerCfg.EncoderConfig.EncodeTime = zapcore.RFC3339TimeEncoder
	logger, _ = loggerCfg.Build()
	defer logger.Sync()
	ctx, cancelCtx = context.WithCancel(context.Background())
	defer cancelCtx()
	if err := envconfig.Process("", cfg); err != nil {
		logger.Fatal("error al cargar las variables de entorno", zap.Error(err))
	}
	storage = discord.NewInMemoryStorage()
	youtubeFetcher = fetcher.NewYoutubeFetcher()

	handler := discord.NewInteractionHandler(ctx, cfg.DiscordToken, youtubeFetcher, storage, cfg).WithLogger(logger.Named("interactionHandler"))
	commandHandler := discord.NewSlashCommandRouter(cfg.CommandPrefix).
		PlayHandler(handler.PlaySong).
		SkipHandler(handler.SkipSong).
		StopHandler(handler.StopPlaying).
		ListHandler(handler.ListPlaylist).
		RemoveHandler(handler.RemoveSong).
		PlayingNowHandler(handler.GetPlayingSong).
		AddSongOrPlaylistHandler(handler.AddSongOrPlaylist)

	dg, err := discordgo.New("Bot " + cfg.DiscordToken)
	if err != nil {
		logger.Fatal("error al crear la session de discord", zap.Error(err))
		return
	}
	dg.AddHandler(handler.Ready)
	dg.AddHandler(handler.GuildCreate)
	dg.AddHandler(handler.GuildDelete)

	dg.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		switch i.Type {
		case discordgo.InteractionMessageComponent:
			if h, ok := commandHandler.GetComponentHandlers()[i.MessageComponentData().CustomID]; ok {
				h(s, i)
			}

		default:
			if h, ok := commandHandler.GetCommandHandlers()[i.ApplicationCommandData().Name]; ok {
				h(s, i)
			}
		}
	})
	dg.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMessages | discordgo.IntentsGuildVoiceStates
	err = dg.Open()
	if err != nil {
		logger.Fatal("error al abrir la session de discord", zap.Error(err))
	}
	defer dg.Close()
	slashCommands := commandHandler.GetSlashCommands()
	registeredCommands, err := dg.ApplicationCommandBulkOverwrite(dg.State.User.ID, cfg.GuildID, slashCommands)
	if err != nil {
		logger.Fatal("no se pudo realizar el comando de sobrescritura masiva", zap.Error(err))
	}
	if cfg.GuildID != "" {
		defer func() {
			for _, cmd := range registeredCommands {
				if err := dg.ApplicationCommandDelete(dg.State.User.ID, cfg.GuildID, cmd.ID); err != nil {
					logger.Error("no se pudo realizar el comando de sobrescritura masiva", zap.String("command", cmd.Name), zap.Error(err))
				}
			}
		}()
	}
	logger.Info("bot esta corriendo. Apreta ctrl - alt para salir")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
}
