package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/mark3labs/mcp-go/server"
	"go.uber.org/zap"

	"github.com/zekker6/mcp-hubspot-go/internal/tools"
	"github.com/zekker6/mcp-hubspot-go/lib/hubspot"
	"github.com/zekker6/mcp-hubspot-go/lib/logger"
)

const (
	envAccessToken = "HUBSPOT_ACCESS_TOKEN"
	envReadOnly    = "HUBSPOT_MCP_READ_ONLY"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var (
	accessToken          = flag.String("access-token", "", "HubSpot private app access token (env: "+envAccessToken+")")
	readOnly             = flag.Bool("read-only", false, "Disable write tools at registration time (env: "+envReadOnly+")")
	mode                 = flag.String("mode", "stdio", "Mode to run the MCP server in (stdio, sse, http)")
	httpListenAddr       = flag.String("httpListenAddr", ":8012", "Address to listen for http connections in sse/http mode")
	heartbeatInterval    = flag.Duration("httpHeartbeatInterval", 30*time.Second, "Interval for sending heartbeat messages. Only used when -mode=http")
	sseKeepAliveInterval = flag.Duration("sseKeepAliveInterval", 30*time.Second, "Interval for sending keep-alive messages. Only used when -mode=sse")
)

// resolveReadOnly returns the effective read-only setting given the parsed
// flag value, whether the flag was set explicitly, and the environment value.
// Flag wins when set explicitly; otherwise the env var is parsed via
// strconv.ParseBool. Garbage env values resolve to false.
func resolveReadOnly(flagValue, flagSet bool, envValue string) bool {
	if flagSet {
		return flagValue
	}
	if envValue == "" {
		return false
	}
	parsed, err := strconv.ParseBool(envValue)
	if err != nil {
		return false
	}
	return parsed
}

func flagWasSet(name string) bool {
	set := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			set = true
		}
	})
	return set
}

func main() {
	flag.Parse()

	if err := logger.Init(); err != nil {
		fmt.Fprintln(os.Stderr, "logger init:", err)
		os.Exit(1)
	}
	defer logger.Stop()

	switch *mode {
	case "stdio", "sse", "http":
	default:
		logger.Error("invalid mode", zap.String("mode", *mode), zap.String("supported", "stdio|sse|http"))
		os.Exit(1)
	}

	if *mode == "sse" || *mode == "http" {
		if *httpListenAddr == "" {
			logger.Error("httpListenAddr must be set in sse/http mode", zap.String("mode", *mode))
			os.Exit(1)
		}
	}

	token := *accessToken
	if token == "" {
		token = os.Getenv(envAccessToken)
	}
	if token == "" {
		logger.Error("HubSpot access token is required", zap.String("flag", "-access-token"), zap.String("env", envAccessToken))
		os.Exit(1)
	}

	effectiveReadOnly := resolveReadOnly(*readOnly, flagWasSet("read-only"), os.Getenv(envReadOnly))

	hsClient, err := hubspot.NewClient(token)
	if err != nil {
		logger.Error("failed to construct HubSpot client", zap.Error(err))
		os.Exit(1)
	}

	s := server.NewMCPServer(
		"hubspot-manager",
		"v0.1.0",
		server.WithToolCapabilities(false),
		server.WithRecovery(),
	)

	toolCount := tools.RegisterTools(s, hsClient, effectiveReadOnly)

	logger.Info("server starting",
		zap.String("version", version),
		zap.String("commit", commit),
		zap.String("date", date),
		zap.String("mode", *mode),
		zap.Bool("read_only", effectiveReadOnly),
		zap.Int("tool_count", toolCount),
	)

	switch *mode {
	case "stdio":
		if err := server.ServeStdio(s); err != nil {
			logger.Error("stdio server failed", zap.Error(err))
			os.Exit(1)
		}
	case "sse":
		sseServer := server.NewSSEServer(s,
			server.WithKeepAliveInterval(*sseKeepAliveInterval),
		)
		if err := sseServer.Start(*httpListenAddr); err != nil {
			logger.Error("sse server failed", zap.Error(err))
			os.Exit(1)
		}
	case "http":
		httpServer := server.NewStreamableHTTPServer(s,
			server.WithHeartbeatInterval(*heartbeatInterval),
		)
		if err := httpServer.Start(*httpListenAddr); err != nil {
			logger.Error("http server failed", zap.Error(err))
			os.Exit(1)
		}
	}
}
