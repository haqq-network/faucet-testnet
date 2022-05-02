package server

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/LK4D4/trylock"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/negroni"

	"github.com/haqq-network/faucet-testnet/internal/chain"
	"github.com/haqq-network/faucet-testnet/internal/server/authenticator"
	"github.com/haqq-network/faucet-testnet/web"

	sessions "github.com/goincremental/negroni-sessions"
	"github.com/goincremental/negroni-sessions/cookiestore"
)

const AddressKey string = "address"

type Server struct {
	chain.TxBuilder
	mutex trylock.Mutex
	cfg   *Config
	queue chan string
}

func NewServer(builder chain.TxBuilder, cfg *Config) *Server {
	return &Server{
		TxBuilder: builder,
		cfg:       cfg,
		queue:     make(chan string, cfg.queueCap),
	}
}

func (s *Server) setupRouter(auth *authenticator.Authenticator) *http.ServeMux {
	router := http.NewServeMux()

	router.Handle("/", http.FileServer(web.Dist()))

	limiter := NewLimiter(s.cfg.proxyCount, time.Duration(s.cfg.interval)*time.Minute)
	router.Handle("/api/claim", negroni.New(limiter, negroni.Wrap(s.handleClaim())))
	router.Handle("/api/info", s.handleInfo())

	router.Handle("/callback", s.callback(auth))
	router.Handle("/login", s.login(auth))

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

	auth, err := authenticator.New()
	if err != nil {
		log.Fatalf("Failed to initialize the authenticator: %v", err)
	}

	store := cookiestore.New([]byte("secret"))
	n.Use(sessions.Sessions("auth-session", store))

	n.UseHandler(s.setupRouter(auth))

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
		Network string `json:"network"`
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
			Network: s.cfg.network,
			Payout:  strconv.Itoa(s.cfg.payout),
		})
	}
}

func (s *Server) login(auth *authenticator.Authenticator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		state, err := generateRandomState()
		if err != nil {
			log.WithError(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Save the state inside the session.
		session := sessions.GetSession(r)
		session.Set("state", state)

		http.Redirect(w, r, auth.AuthCodeURL(state), http.StatusTemporaryRedirect)
	}
}

func (s *Server) callback(auth *authenticator.Authenticator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session := sessions.GetSession(r)

		fmt.Println(session.Get("state"))
		fmt.Println(r.URL.Query().Get("state"))

		//if session.Get("state") != r.URL.Query().Get("state") {
		//	log.Error("Invalid state parameter.")
		//	http.Error(w, "Invalid state parameter.", http.StatusBadRequest)
		//	return
		//}

		// Exchange an authorization code for a token.
		fmt.Println(r.URL.Query().Get("code"))
		token, err := auth.Exchange(context.Background(), r.URL.Query().Get("code"))
		if err != nil {
			log.WithError(err)
			http.Error(w, "Failed to exchange an authorization code for a token.", http.StatusUnauthorized)
			return
		}

		idToken, err := auth.VerifyIDToken(context.Background(), token)
		if err != nil {
			log.WithError(err)
			http.Error(w, "Failed to verify ID Token.", http.StatusInternalServerError)
			return
		}

		var profile map[string]interface{}
		if err := idToken.Claims(&profile); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		session.Set("access_token", token.AccessToken)
		session.Set("profile", profile)

		// Redirect to logged in page.
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	}
}

func generateRandomState() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}

	state := base64.StdEncoding.EncodeToString(b)

	return state, nil
}
