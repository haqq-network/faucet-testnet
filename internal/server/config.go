package server

type Config struct {
	httpPort   int
	interval   int
	payout     int
	proxyCount int
	queueCap   int
}

func NewConfig(httpPort, interval, payout, proxyCount, queueCap int) *Config {
	return &Config{
		httpPort:   httpPort,
		interval:   interval,
		payout:     payout,
		proxyCount: proxyCount,
		queueCap:   queueCap,
	}
}
