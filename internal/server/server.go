package server

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-pg/pg"
	"github.com/haqq-network/faucet-testnet/database"
	"github.com/spf13/viper"
	"net/http"
	"strconv"
	"time"

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
	cfg          Config
	queue        chan string
	requestStore *database.RequestStore
	db           *pg.DB
}

func NewServer(builder chain.TxBuilder) *Server {

	db, err := database.DBConn()
	if err != nil {
		panic(err.Error())
		return nil
	}

	requestStore := database.NewRequestStore(db)

	cfg := Config{
		httpPort:   viper.GetInt("httpPort"),
		interval:   viper.GetInt("interval"),
		payout:     viper.GetInt("payout"),
		proxyCount: viper.GetInt("proxyCount"),
		queueCap:   viper.GetInt("queueCap"),
	}
	return &Server{
		TxBuilder:    builder,
		cfg:          cfg,
		queue:        make(chan string, cfg.queueCap),
		requestStore: requestStore,
		db:           db,
	}
}

func (s *Server) setupRouter() *http.ServeMux {
	router := http.NewServeMux()

	router.Handle("/", http.FileServer(web.Dist()))

	limiter := NewLimiter(s.cfg.proxyCount, time.Duration(s.cfg.interval)*time.Minute)
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

	n := negroni.New(negroni.NewRecovery(), negroni.NewLogger())
	n.UseHandler(s.setupRouter())
	log.Infof("Starting http server %d", s.cfg.httpPort)
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(s.cfg.httpPort), n))
}

func (s *Server) consumeQueue() {
	if len(s.queue) == 0 {
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()
	for len(s.queue) != 0 {
		address := <-s.queue
		txHash, err := s.Transfer(context.Background(), address, chain.EtherToWei(int64(s.cfg.payout)))
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
		_, err := s.requestStore.Insert(github)
		if err != nil {
			log.WithError(err).Error("Failed to save request")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		txHash, err := s.Transfer(ctx, address, chain.EtherToWei(int64(s.cfg.payout)))
		s.mutex.Unlock()
		if err != nil {
			log.WithError(err).Error("Failed to send transaction")
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
			Payout:  strconv.Itoa(s.cfg.payout),
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
