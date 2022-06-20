package cmd

import (
	"fmt"
	"math/big"
	"os"
	"os/signal"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/haqq-network/faucet-testnet/internal/chain"
	"github.com/haqq-network/faucet-testnet/internal/server"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	RootCmd.AddCommand(serveCmd)

	// Here you will define your flags and configuration settings.
	viper.SetDefault("httpport", 8080)
	viper.SetDefault("proxycount", 0)
	viper.SetDefault("queuecap", 100)
	viper.SetDefault("amount", 1)
	viper.SetDefault("interval", 1440)
	viper.SetDefault("chainID", 53211)
}

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "start http server with configured api",
	Long:  `Starts a http server and serves the configured api`,
	Run: func(cmd *cobra.Command, args []string) {

		privateKey, err := crypto.HexToECDSA(os.Getenv("PRIVATE_KEY"))
		if err != nil {
			panic(fmt.Errorf("failed to read private key: %w", err))
		}

		txBuilder, err := chain.NewTxBuilder(os.Getenv("WEB3_PROVIDER"), privateKey, big.NewInt(viper.GetInt64("chainID")))
		if err != nil {
			panic(fmt.Errorf("cannot connect to web3 provider: %w", err))
		}

		go server.NewServer(txBuilder).Run()

		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		<-c
	},
}

//package cmd
//
//import (
//	"crypto/ecdsa"
//	"errors"
//	"flag"
//	"fmt"
//	"math/big"
//	"os"
//	"os/signal"
//	"strings"
//
//	"github.com/joho/godotenv"
//
//	"github.com/ethereum/go-ethereum/crypto"
//
//	"github.com/haqq-network/faucet-testnet/internal/chain"
//	"github.com/haqq-network/faucet-testnet/internal/server"
//)
//

//

//
//func Execute() {
//	if err := godotenv.Load(); err != nil {
//		panic(fmt.Errorf("Failed to load the env vars: %v", err))
//	}
//
//	privateKey, err := getPrivateKeyFromFlags()
//	if err != nil {
//		panic(fmt.Errorf("failed to read private key: %w", err))
//	}
//	var chainID *big.Int
//	if value, ok := chainIDMap[strings.ToLower(*netnameFlag)]; ok {
//		chainID = big.NewInt(int64(value))
//	}
//
//	txBuilder, err := chain.NewTxBuilder(os.Getenv("WEB3_PROVIDER"), privateKey, chainID)
//	if err != nil {
//		panic(fmt.Errorf("cannot connect to web3 provider: %w", err))
//	}
//	config := server.NewConfig(*netnameFlag, *httpPortFlag, *intervalFlag, *payoutFlag, *proxyCntFlag, *queueCapFlag)
//	go server.NewServer(txBuilder, config).Run()
//
//	c := make(chan os.Signal, 1)
//	signal.Notify(c, os.Interrupt)
//	<-c
//}
//
