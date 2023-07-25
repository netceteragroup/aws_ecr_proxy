package main

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/terrycain/aws_ecr_proxy/internal/ecr_token"
	"github.com/terrycain/aws_ecr_proxy/internal/proxy_server"
	"github.com/terrycain/aws_ecr_proxy/internal/utils"
	"github.com/terrycain/aws_ecr_proxy/internal/version"
)

func main() {
	envLevel := utils.GetEnv("LOG_LEVEL", "INFO")
	zerolog.SetGlobalLevel(utils.LogNameToLevel(envLevel))

	disableProxyHeaders := utils.GetEnv("DISABLE_PROXY_HEADERS", "false") == "true"
	host := utils.GetEnv("LISTEN_HOST", "0.0.0.0")
	port := utils.GetEnv("LISTEN_PORT", "8080")
	addr := host + ":" + port

	log.Info().Str("version", version.VERSION).Str("build_date", version.BUILDDATE).Str("sha", version.SHA).Msg("Starting ECR Proxy")
	awsSession, err := session.NewSession()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create an AWS session, check credentials")
		panic(err)
	}

	awsCreds := &credentials.Credentials{}

	// Optionally, assume a role
	roleToAssume := utils.GetEnv("ASSUME_ROLE", "")
	if len(roleToAssume) > 0 {
		log.Info().Msgf("Assuming role %s", roleToAssume)
		awsCreds = stscreds.NewCredentials(awsSession, roleToAssume)
		_, err := awsCreds.Get()
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to assume role")
			panic(err)
		}
	} else {
		awsCreds = awsSession.Config.Credentials
	}

	// Create an ECR client
	svc := ecr.New(awsSession, &aws.Config{Credentials: awsCreds})

	// Instantiate our ecs token getter
	tokenFetcher := ecr_token.New(svc)
	go tokenFetcher.Run()
	defer tokenFetcher.Close()

	// Pass reference to our refreshing token to the HTTP server
	proxy_server.Run(addr, disableProxyHeaders, tokenFetcher)
}
