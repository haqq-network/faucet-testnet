package server

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-contrib/static"
	"github.com/go-pg/pg"
	"github.com/haqq-network/faucet-testnet/database"
	"github.com/haqq-network/faucet-testnet/internal/authenticator"
	"github.com/haqq-network/faucet-testnet/internal/middleware"
	"github.com/joho/godotenv"
	"github.com/spf13/viper"

	"github.com/gin-gonic/gin"

	"github.com/LK4D4/trylock"
	log "github.com/sirupsen/logrus"

	"github.com/haqq-network/faucet-testnet/internal/chain"
)

const AddressKey string = "address"
const GithubKey string = "github"
const UserId string = "user_id"

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

func (s *Server) setupRouter(auth *authenticator.Authenticator) *gin.Engine {
	router := gin.Default()
	gob.Register(map[string]interface{}{})

	store := cookie.NewStore([]byte("secret"))
	router.Use(sessions.Sessions("auth-session", store))

	router.Use(static.Serve("/", static.LocalFile("./web/public", true)))

	api := router.Group("/api")
	{
		api.GET("/login", HandlerLogin(auth))
		api.GET("/callback", HandlerCallback(auth))
		api.POST("/claim", middleware.IsAuthenticated, s.handleClaim())
		api.GET("/info", middleware.IsAuthenticated, s.handleInfo())
		api.GET("/requested", middleware.IsAuthenticated, s.handleLastRequest())
		api.GET("/logout", HandlerLogout)
	}

	return router
}

func (s *Server) Run() {
	go func() {
		ticker := time.NewTicker(time.Second)
		for range ticker.C {
			s.consumeQueue()
		}
	}()

	if err := godotenv.Load(); err != nil {
		log.Fatalf("Failed to load the env vars: %v", err)
	}

	auth, err := authenticator.New()
	if err != nil {
		log.Fatalf("Failed to initialize the authenticator: %v", err)
	}

	rtr := s.setupRouter(auth)

	log.Print("Server listening on http://localhost:" + viper.GetString("httpport"))
	if err := http.ListenAndServe(":"+viper.GetString("httpport"), rtr); err != nil {
		log.Fatalf("There was an error with the http server: %v", err)
	}
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

func (s *Server) handleClaim() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		address := ctx.Request.PostFormValue(AddressKey)
		re := regexp.MustCompile("^0x[0-9a-fA-F]{40}$")
		if !re.MatchString(address) {
			ctx.String(http.StatusInternalServerError, "Invalid address")
			return
		}

		session := sessions.Default(ctx)
		profile := session.Get("profile")

		if profile == nil {
			ctx.String(http.StatusInternalServerError, "You need to login first")
			return
		}

		github := profile.(map[string]interface{})["nickname"].(string)

		log.WithFields(log.Fields{
			"address": address,
			"github":  github,
		}).Info("Received request")

		_, err := s.requestStore.Insert(github, address)
		if err != nil {
			log.WithError(err).Error("Failed to save request")
			ctx.String(http.StatusInternalServerError, err.Error())
			return
		}

		// Try to lock mutex if the work queue is empty
		if len(s.queue) != 0 || !s.mutex.TryLock() {
			select {
			case s.queue <- address:
				log.WithFields(log.Fields{
					"address": address,
				}).Info("Added to queue successfully")
				fmt.Fprintf(ctx.Writer, "Added %s to the queue", address)
			default:
				log.Warn("Max queue capacity reached")
				errMsg := "Faucet queue is too long, please try again later"
				ctx.String(http.StatusServiceUnavailable, errMsg)
				return
			}
			return
		}

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
			ctx.String(http.StatusServiceUnavailable, "Failed to send transaction")
			return
		}

		log.WithFields(log.Fields{
			"txHash":  txHash,
			"address": address,
		}).Info("Funded directly successfully")
		ctx.String(http.StatusOK, "Tx hash: %s", txHash)
	}
}

func (s *Server) handleInfo() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		session := sessions.Default(ctx)
		profile := session.Get("profile")
		ctx.JSON(http.StatusOK, profile)
	}
}

type request struct {
	Github            string `json:"github"`
	LastRequestedTime int64  `json:"last_requested_time"`
}

func (s *Server) handleLastRequest() gin.HandlerFunc {
	return func(ctx *gin.Context) {

		github := ctx.Request.FormValue(GithubKey)
		if len(github) == 0 {
			ctx.String(http.StatusInternalServerError, "Empty github name")
			return
		}

		req, err := s.requestStore.Get(github)
		if err != nil {
			if err.Error() == "pg: no rows in result set" {
				ctx.String(http.StatusNotFound, "Account not found")
				return
			}
			ctx.String(http.StatusInternalServerError, err.Error())
			return
		}

		ctx.Writer.Header().Set("Content-Type", "application/json")
		json.NewEncoder(ctx.Writer).Encode(request{
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
