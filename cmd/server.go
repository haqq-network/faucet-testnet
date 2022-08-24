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
	viper.SetDefault("amount", 1000)
	viper.SetDefault("interval", 1440)
	viper.SetDefault("chain", 53211)
	viper.SetDefault("slack_hook", "https://hooks.slack.com/services/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")
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

		txBuilder, err := chain.NewTxBuilder(os.Getenv("WEB3_PROVIDER"), privateKey, big.NewInt(viper.GetInt64("chain")))
		if err != nil {
			panic(fmt.Errorf("cannot connect to web3 provider: %w", err))
		}

		go server.NewServer(txBuilder).Run()

		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		<-c
	},
}
