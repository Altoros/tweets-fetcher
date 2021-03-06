package main

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/cloudfoundry-community/go-cfenv"
	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	log "github.com/inconshreveable/log15"
	"github.com/quipo/statsd"
	"googlemaps.github.io/maps"

	"github.com/Altoros/tweets-fetcher/fetcher"
	"github.com/Altoros/tweets-fetcher/geocoder"
	"github.com/Altoros/tweets-fetcher/server"
)

var (
	defaultPort          = "8080"
	defaultLogLevel      = log.LvlInfo
	requiredEnvVariables = []string{
		"TWITTER_CONSUMER_KEY",
		"TWITTER_CONSUMER_SECRET",
		"TWITTER_CONSUMER_ACCESS_TOKEN",
		"TWITTER_CONSUMER_ACCESS_SECRET",
	}

	statsdServiceName = os.Getenv("CF_MONITORING_SERVICE_NAME")
)

func main() {
	if statsdServiceName == "" {
		statsdServiceName = "heartbeat"
	}

	logger := log.New("module", "main")
	logger.SetHandler(log.LvlFilterHandler(getloggerLvl(), log.StreamHandler(os.Stdout, log.JsonFormat())))

	var err error

	err = checkReqiredEnvVariables()
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	statsdClient := statsdClient(logger)
	err = statsdClient.CreateSocket()
	if err != nil {
		logger.Error("Failed to create statsd socket", "err", err)
		os.Exit(1)
	}

	twitterClient := twitterClient(
		os.Getenv("TWITTER_CONSUMER_KEY"),
		os.Getenv("TWITTER_CONSUMER_SECRET"),
		os.Getenv("TWITTER_CONSUMER_ACCESS_TOKEN"),
		os.Getenv("TWITTER_CONSUMER_ACCESS_SECRET"),
	)

	fetcher := fetcher.New(logger, twitterClient, statsdClient, geoCoder(logger))

	server := server.New(logger, fetcher)
	errChan := make(chan error)
	go server.Start(errChan, getPort())

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, os.Kill, syscall.SIGTERM)

	select {
	case err := <-errChan:
		logger.Error("Start server", "err", err)
	case sig := <-signalChan:
		logger.Info(fmt.Sprintf("Received signal %s. shutting down", sig))
		server.Stop()
	}
}

func getPort() string {
	if os.Getenv("PORT") == "" {
		return defaultPort
	}
	return os.Getenv("PORT")
}

func getloggerLvl() log.Lvl {
	levelStr := os.Getenv("LOG_LEVEL")
	lvl, err := log.LvlFromString(levelStr)
	if err != nil {
		return defaultLogLevel
	}
	return lvl
}

func checkReqiredEnvVariables() error {
	for _, variable := range requiredEnvVariables {
		if os.Getenv(variable) == "" {
			return fmt.Errorf("Env variable %s must be set", variable)
		}
	}

	if os.Getenv("GOOGLE_MAPS_KEY") == "" && os.Getenv("BING_MAPS_KEY") == "" {
		return errors.New("Either GOOGLE_MAPS_KEY or BING_MAPS_KEY env variable should be set")
	}

	return nil
}

func statsdClient(logger log.Logger) statsd.Statsd {
	appEnv, err := cfenv.Current()
	if err != nil {
		logger.Warn("Failed to get CF env, continuing with noop statsd client", "err", err)
		return &statsd.NoopClient{}
	}
	service, err := appEnv.Services.WithName(statsdServiceName)
	if err != nil {
		logger.Warn(fmt.Sprintf("Couldn't find statsd service '%s', continuing with noop statsd client", statsdServiceName), "err", err)
		return &statsd.NoopClient{}
	}

	host := service.Credentials["host"]
	port := service.Credentials["port"]
	statsdPrefix := service.Credentials["statsd_prefix"]

	addr := fmt.Sprintf("%v:%v", host, port)
	prefix := fmt.Sprintf("%s%s.%d.", statsdPrefix, appEnv.Name, appEnv.Index)

	return statsd.NewStatsdClient(addr, prefix)
}

func twitterClient(consumerKey, consumerSecret, accessToken, accessSecret string) *twitter.Client {
	config := oauth1.NewConfig(consumerKey, consumerSecret)
	token := oauth1.NewToken(accessToken, accessSecret)
	httpClient := config.Client(oauth1.NoContext, token)
	return twitter.NewClient(httpClient)
}

func geoCoder(logger log.Logger) geocoder.Geocoder {
	if os.Getenv("BING_MAPS_KEY") != "" {
		logger.Info("Using Bing maps to geocode")
		return geocoder.NewBing(os.Getenv("BING_MAPS_KEY"))
	} else {
		logger.Info("Using Google maps to geocode")
		googleMapsClient, err := maps.NewClient(maps.WithAPIKey(os.Getenv("GOOGLE_MAPS_KEY")))
		if err != nil {
			logger.Error("Failed to create google maps API client", "err", err)
			os.Exit(1)
		}
		return geocoder.NewGoogle(googleMapsClient)
	}

	return nil
}
