package handlers_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"

	log "github.com/inconshreveable/log15"

	"github.com/Altoros/tweets-fetcher/fetcher"
	"github.com/Altoros/tweets-fetcher/server/handlers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type fakeFetcher struct {
	query string
}

func (ff *fakeFetcher) Fetch(query string) {
	ff.query = query
}

func (ff *fakeFetcher) Stop() {
	ff.query = ""
}

func (ff *fakeFetcher) Tweets() chan *fetcher.Tweet {
	return make(chan *fetcher.Tweet)
}

func (ff *fakeFetcher) CurrentQuery() string {
	return ff.query
}

type fakeFanout struct {
}

func (ffo *fakeFanout) Register(client *handlers.Client) {
}

func (ffo *fakeFanout) Unregister(client *handlers.Client) {
}

func (ffo *fakeFanout) Run() {
}

func (ffo *fakeFanout) UnregisterAll() {
}

var _ = Describe("Fetcher handlers", func() {
	var api http.Handler
	fetcher := &fakeFetcher{}
	fanout := &fakeFanout{}

	BeforeEach(func() {
		logger := log.New()
		logger.SetHandler(log.DiscardHandler())
		api = handlers.New(logger, fetcher, fanout, "../../templates")
	})

	Describe("home", func() {
		It("returns MethodNotAllowed if not GET", func() {
			req, err := http.NewRequest("POST", "/", nil)
			Expect(err).NotTo(HaveOccurred())

			rr := httptest.NewRecorder()
			api.ServeHTTP(rr, req)

			Ω(rr.Code).Should(Equal(http.StatusMethodNotAllowed))
		})

		It("returns homepage", func() {
			req, err := http.NewRequest("GET", "/", nil)
			Expect(err).NotTo(HaveOccurred())

			rr := httptest.NewRecorder()
			api.ServeHTTP(rr, req)

			Ω(rr.Code).Should(Equal(http.StatusOK))
		})
	})

	Describe("query", func() {
		It("returns query", func() {
			req, err := http.NewRequest("GET", "/query", nil)
			Expect(err).NotTo(HaveOccurred())

			fetcher.query = "test"

			rr := httptest.NewRecorder()
			api.ServeHTTP(rr, req)

			Expect(rr.Body.String()).To(Equal("test"))
		})
	})

	Describe("fetch", func() {
		It("returns 400 if no query provided", func() {
			req, err := http.NewRequest("POST", "/fetch", nil)
			Expect(err).NotTo(HaveOccurred())

			rr := httptest.NewRecorder()
			api.ServeHTTP(rr, req)

			Ω(rr.Code).Should(Equal(http.StatusBadRequest))
			Expect(rr.Body.String()).To(Equal("Request body is empty\n"))
		})

		It("returns 400 if query is empty", func() {
			buffer := &bytes.Buffer{}
			buffer.WriteString("")
			req, err := http.NewRequest("POST", "/fetch", buffer)
			Expect(err).NotTo(HaveOccurred())

			rr := httptest.NewRecorder()
			api.ServeHTTP(rr, req)

			Ω(rr.Code).Should(Equal(http.StatusBadRequest))
			Expect(rr.Body.String()).To(Equal("Query can't be blank\n"))
		})

		It("updates fetcher's current query if query specified", func() {
			buffer := &bytes.Buffer{}
			buffer.WriteString("query")
			req, err := http.NewRequest("POST", "/fetch", buffer)
			Expect(err).NotTo(HaveOccurred())

			rr := httptest.NewRecorder()
			api.ServeHTTP(rr, req)

			Ω(rr.Code).Should(Equal(http.StatusOK))
			Expect(fetcher.query).To(Equal("query"))
		})
	})
})
