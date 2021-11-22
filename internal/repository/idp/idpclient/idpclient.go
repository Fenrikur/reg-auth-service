package idpclient

import (
	"context"
	"fmt"
	"github.com/eurofurence/reg-auth-service/internal/repository/config"
	"github.com/eurofurence/reg-auth-service/internal/repository/idp"
	"github.com/eurofurence/reg-auth-service/internal/repository/logging"
	"github.com/eurofurence/reg-auth-service/internal/repository/util/downstreamcall"
	"github.com/eurofurence/reg-auth-service/web/util/media"
	"net/http"
	"net/url"
)

type IdentityProviderClientImpl struct {
	netClient *http.Client
}

const CommandName = "idp_token"

// --- instance creation ---

func New() idp.IdentityProviderClient {
	timeout := config.TokenRequestTimeout()

	downstreamcall.ConfigureGobreakerCommand(CommandName)

	return &IdentityProviderClientImpl{
		netClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// --- implementation of repository interface ---

// can leave out fields to demo tolerant reader

type ErrorDto struct {
	ErrorCode        string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

func TokenRequestBody(appConfig config.ApplicationConfig, authorizationCode string, pkceVerifier string) string {
	parameters := url.Values{}
	parameters.Set("grant_type", "authorization_code")
	parameters.Set("client_id", appConfig.ClientId)
	parameters.Set("client_secret", appConfig.ClientSecret)
	parameters.Set("redirect_uri", appConfig.DefaultRedirectUrl)
	parameters.Set("code", authorizationCode)
	parameters.Set("code_verifier", pkceVerifier)
	requestBody := parameters.Encode()
	return requestBody
}

func (i *IdentityProviderClientImpl) TokenWithAuthenticationCodeAndPKCE(ctx context.Context, applicationConfigName string, authorizationCode string, pkceVerifier string) (*idp.TokenResponseDto, error) {
	appConfig, err := config.GetApplicationConfig(applicationConfigName)
	if err != nil {
		logging.Ctx(ctx).Warn(err.Error())
		return nil, err
	}

	requestBody := TokenRequestBody(appConfig, authorizationCode, pkceVerifier)

	tokenEndpoint := config.TokenEndpoint()

	responseBody, httpstatus, err := downstreamcall.PerformPOST(ctx, i.netClient, tokenEndpoint, requestBody, media.ContentTypeApplicationXWwwFormUrlencoded)
	// responseBody, httpstatus, err := downstreamcall.GobreakerPerformPOST(ctx, i.netClient, tokenEndpoint, requestBody, media.ContentTypeApplicationXWwwFormUrlencoded)

	if err != nil || httpstatus != http.StatusOK {
		if err == nil {
			err = fmt.Errorf("unexpected http status %d, was expecting %d", httpstatus, http.StatusOK)
		}

		errorResponseDto := &ErrorDto{}
		err2 := downstreamcall.ParseJson(responseBody, errorResponseDto)
		if err2 == nil {
			logging.Ctx(ctx).Error(fmt.Sprintf("error requesting token from identity provider: error from response is %s:%s, local error is %s", errorResponseDto.ErrorCode, errorResponseDto.ErrorDescription, err.Error()))
		} else {
			logging.Ctx(ctx).Error(fmt.Sprintf("error requesting token from identity provider with no structured response available: local error is %s", err.Error()))
		}

		return nil, err
	}

	successResponseDto := &idp.TokenResponseDto{}
	err = downstreamcall.ParseJson(responseBody, successResponseDto)
	if err != nil {
		logging.Ctx(ctx).Error(fmt.Sprintf("error parsing token response from identity provider: error is %s", err.Error()))
		return nil, err
	}

	return successResponseDto, nil
}
