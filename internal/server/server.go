package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-pg/pg"
	"github.com/haqq-network/faucet-testnet/database"
	"github.com/rs/cors"
	"github.com/spf13/viper"

	"github.com/LK4D4/trylock"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/negroni"

	"github.com/haqq-network/faucet-testnet/internal/chain"
	"github.com/haqq-network/faucet-testnet/web"
)

const AddressKey string = "address"
const GithubKey string = "github"

type Server struct {
	chain.TxBuilder
	mutex        trylock.Mutex
	queue        chan string
	requestStore *database.RequestStore
	db           *pg.DB
}

func NewServer(builder chain.TxBuilder) *Server {

	db, err := database.DBConn()
	if err != nil {
		panic(err.Error())
	}

	requestStore := database.NewRequestStore(db)

	return &Server{
		TxBuilder:    builder,
		queue:        make(chan string, viper.GetInt("queuecap")),
		requestStore: requestStore,
		db:           db,
	}
}

func (s *Server) setupRouter() *http.ServeMux {
	router := http.NewServeMux()

	router.Handle("/", http.FileServer(web.Dist()))

	limiter := NewLimiter(viper.GetInt("proxycount"), time.Duration(viper.GetInt("interval"))*time.Minute)
	router.Handle("/api/claim", negroni.New(limiter, negroni.Wrap(s.handleClaim())))
	router.Handle("/api/info", s.handleInfo())
	router.Handle("/api/requested", s.handleLastRequest())

	return router
}

func (s *Server) Run() {
	go func() {
		ticker := time.NewTicker(time.Second)
		for range ticker.C {
			s.consumeQueue()
		}
	}()

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"https://testedge.haqq.network/"},
		AllowCredentials: true,
		// Enable Debugging for testing, consider disabling in production
		Debug: true,
	})

	n := negroni.New(negroni.NewRecovery(), negroni.NewLogger())
	n.UseHandler(s.setupRouter())
	log.Infof("Starting http server %d", viper.GetInt("httpport"))
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(viper.GetInt("httpport")), c.Handler(n)))
}

func (s *Server) consumeQueue() {
	if len(s.queue) == 0 {
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()
	for len(s.queue) != 0 {
		address := <-s.queue
		txHash, err := s.Transfer(context.Background(), address, chain.EtherToWei(viper.GetInt64("amount")))
		if err != nil {
			log.WithError(err).Error("Failed to handle transaction in the queue")
		} else {
			log.WithFields(log.Fields{
				"txHash":  txHash,
				"address": address,
			}).Info("Consume from queue successfully")
		}
	}
}

func (s *Server) handleClaim() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.NotFound(w, r)
			return
		}

		address := r.PostFormValue(AddressKey)
		github := r.PostFormValue(GithubKey)
		// TODO: check if user has valid github account
		if len(github) == 0 {
			http.Error(w, "github account not valid", http.StatusInternalServerError)
			return
		}
		// Try to lock mutex if the work queue is empty
		if len(s.queue) != 0 || !s.mutex.TryLock() {
			select {
			case s.queue <- address:
				log.WithFields(log.Fields{
					"address": address,
				}).Info("Added to queue successfully")
				fmt.Fprintf(w, "Added %s to the queue", address)
			default:
				log.Warn("Max queue capacity reached")
				errMsg := "Faucet queue is too long, please try again later"
				http.Error(w, errMsg, http.StatusServiceUnavailable)
			}
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		txHash, err := s.Transfer(ctx, address, chain.EtherToWei(viper.GetInt64("amount")))
		s.mutex.Unlock()
		if err != nil {
			go func() {
				if err.Error() == "insufficient funds for gas * price + value" {
					err := SendSlackNotification(fmt.Sprintf("TestEdge Faucet: No funds left for distribution. Please recharge address: %s", s.TxBuilder.Sender().String()))
					if err != nil {
						log.WithError(err).Error("Failed to send notification to slack")
					}
				}
			}()
			log.WithError(err).Error("Failed to send transaction")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		_, err = s.requestStore.Insert(github)
		if err != nil {
			log.WithError(err).Error("Failed to save request")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		log.WithFields(log.Fields{
			"txHash":  txHash,
			"address": address,
		}).Info("Funded directly successfully")
		fmt.Fprintf(w, "Txhash: %s", txHash)
	}
}

func (s *Server) handleInfo() http.HandlerFunc {
	type info struct {
		Account string `json:"account"`
		Payout  string `json:"payout"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(info{
			Account: s.Sender().String(),
			Payout:  strconv.Itoa(viper.GetInt("amount")),
		})
	}
}

func (s *Server) handleLastRequest() http.HandlerFunc {
	type request struct {
		Github            string `json:"github"`
		LastRequestedTime int64  `json:"last_requested_time"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.NotFound(w, r)
			return
		}

		github := r.FormValue(GithubKey)
		if len(github) == 0 {
			http.Error(w, "Empty github name", http.StatusInternalServerError)
			return
		}

		req, err := s.requestStore.Get(github)
		if err != nil {
			if err.Error() == "pg: no rows in result set" {
				http.Error(w, "Account not found", http.StatusNotFound)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(request{
			Github:            req.Github,
			LastRequestedTime: req.RequestDate,
		})
	}
}

// SendSlackNotification will post to an 'Incoming Webook' url setup in Slack Apps. It accepts
// some text and the slack channel is saved within Slack.
func SendSlackNotification(msg string) error {
	type SlackRequestBody struct {
		Text string `json:"text"`
	}

	slackBody, _ := json.Marshal(SlackRequestBody{Text: msg})
	req, err := http.NewRequest(http.MethodPost, viper.GetString("slack_hook"), bytes.NewBuffer(slackBody))
	if err != nil {
		return err
	}

	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	if buf.String() != "ok" {
		return errors.New("non-ok response returned from Slack")
	}
	return nil
}
