package fetcher

import (
	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	log "github.com/inconshreveable/log15"
	"github.com/quipo/statsd"
)

type fetcher struct {
	logger        log.Logger
	query         string
	twitterClient *twitter.Client
	currentStream *twitter.Stream
	tweets        chan *Tweet
	statsdClient  statsd.Statsd
}

type Fetcher interface {
	Fetch(string)
	Stop()
	Tweets() chan *Tweet
	CurrentQuery() string
}

func New(logger log.Logger, statsdClient statsd.Statsd, opts Options) Fetcher {
	config := oauth1.NewConfig(opts.TwitterConsumerKey, opts.TwitterConsumerSecret)
	token := oauth1.NewToken(opts.TwitterAccessToken, opts.TwitterAccessSecret)
	httpClient := config.Client(oauth1.NoContext, token)
	twitterClient := twitter.NewClient(httpClient)

	return &fetcher{
		logger:        logger.New("module", "fetcher"),
		twitterClient: twitterClient,
		tweets:        make(chan *Tweet),
		statsdClient:  statsdClient,
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
		for message := range stream.Messages {
			switch v := message.(type) {
			case *twitter.Tweet:
				err := f.statsdClient.Incr("totalTweets", 1)
				if err != nil {
					f.logger.Warn("Failed to emit metric", "err", err)
				}
				tweet := v
				if tweet.Coordinates != nil {
					f.logger.Debug("Received a tweet", "text", tweet.Text, "coordinates", tweet.Coordinates.Coordinates)
					f.tweets <- &Tweet{
						Id:   tweet.IDStr,
						Text: tweet.Text,
						User: tweet.User.ScreenName,
						Coordinates: Coordinates{
							Long: tweet.Coordinates.Coordinates[0],
							Lat:  tweet.Coordinates.Coordinates[1],
						},
					}
					err := f.statsdClient.Incr("tweetsWithLocation", 1)
					if err != nil {
						f.logger.Warn("Failed to emit metric", "err", err)
					}
				} else {
					f.logger.Debug("Received a tweet without location, skipping")
				}
			case *twitter.StreamLimit:
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
