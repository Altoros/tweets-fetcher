# tweets-fetcher - demo app to demonstrate sending of custom metrics by using statsd

## Deploy

Take [manifest.yml](ci/manifest/manifest.yml) set the following env vars in it:
```
BING_MAPS_KEY: xxx
GOOGLE_MAPS_KEY: xxx
TWITTER_CONSUMER_ACCESS_SECRET: xxx
TWITTER_CONSUMER_ACCESS_TOKEN: xxx
TWITTER_CONSUMER_KEY: xxx
TWITTER_CONSUMER_SECRET: xxx
```
Then just `cf push` this app!

## Running tests

```
./bin/test
```

## CI

[There is concourse pipeline!](https://concourse.altoros.com/teams/main/pipelines/cf-tweets-fetcher-app?groups=cf-tweets-fetcher-app)
