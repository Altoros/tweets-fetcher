package fetcher

import (
	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	log "github.com/inconshreveable/log15"
)

type fetcher struct {
	logger        log.Logger
	query         string
	twitterClient *twitter.Client
	currentStream *twitter.Stream
	tweets        chan *Tweet
}

type Fetcher interface {
	Fetch(string)
	Stop()
	Tweets() chan *Tweet
	CurrentQuery() string
}

func New(logger log.Logger) Fetcher {
	config := oauth1.NewConfig("MHba5z4mb4l5kePtWrq9LnaIF", "40U9IO7daGfRyCmxRNKeRfhWFyto8PsVllg8exmbDtLxF76tQf")
	token := oauth1.NewToken("96115591-MvlX96UkkLKolQhtOs3b8gVkZ1Y7vXMGxTLyGDYHS", "mnF46bAeQj9GjtSeFww94Ak37quWj2RRoVrakyy39FPpI")
	httpClient := config.Client(oauth1.NoContext, token)
	twitterClient := twitter.NewClient(httpClient)

	return &fetcher{
		logger:        logger.New("module", "fetcher"),
		twitterClient: twitterClient,
		tweets:        make(chan *Tweet),
	}
}

func (f *fetcher) Fetch(query string) {
	f.logger.Info("Fetch request", "query", query)

	f.stopFetching()
	f.query = query
	err := f.startFetching()
	if err != nil {
		f.logger.Error("Fetching tweets", "err", err)
		return
	}

	go func(stream *twitter.Stream) {
		for untypedMessage := range stream.Messages {
			switch v := untypedMessage.(type) {
			case *twitter.Tweet:
				message := v
				if message.Coordinates != nil {
					f.logger.Debug("Received a tweet", "text", message.Text, "coordinates", message.Coordinates.Coordinates)
					f.tweets <- &Tweet{
						Id:   message.IDStr,
						Text: message.Text,
						User: message.User.ScreenName,
						Coordinates: Coordinates{
							Long: message.Coordinates.Coordinates[0],
							Lat:  message.Coordinates.Coordinates[1],
						},
					}
				}
			case *twitter.StreamLimit:
				panic("!!!")
				f.logger.Warn("Stream limit", "track", v.Track)
			}
		}
	}(f.currentStream)
}

func (f *fetcher) Stop() {
	f.stopFetching()
	f.query = ""
}

func (f *fetcher) Tweets() chan *Tweet {
	return f.tweets
}

func (f *fetcher) stopFetching() {
	if f.currentStream != nil {
		f.logger.Info("Stop fetching", "query", f.query)
		f.currentStream.Stop()
		f.currentStream = nil
	}
}

func (f *fetcher) startFetching() error {
	f.logger.Info("Start fetching", "query", f.query)
	params := &twitter.StreamFilterParams{
		Track:         []string{f.query},
		StallWarnings: twitter.Bool(true),
	}
	stream, err := f.twitterClient.Streams.Filter(params)
	if err != nil {
		return err
	}
	f.currentStream = stream
	return nil
}

func (f *fetcher) CurrentQuery() string {
	return f.query
}
