package middleware

import (
	"context"
	"net/http"

	"github.com/rs/zerolog/log"
)

const (
	// TODO(maybe): could just use 'Accept-Language' header, with example content [en-US,en;q=0.9]
	httpHeaderLanguage = "Language"
	contextLanguage    = "Language"

	httpHeaderAuthenticationToken = "Authentication"
	contextAuthenticationToken    = "Authentication"

	httpHeaderUserID = "UserID"
	contextUserID    = "UserID"
)

func AddAll(next http.Handler) http.Handler {
	return AddUserID(AddAuthentication(AddLanguageAndLogging(next)))
}

func AddLanguageAndLogging(next http.Handler) http.Handler {
	return translateHTTPHeaderToContextValue(next, headerConfig{
		Name:        "language",
		HTTPHeader:  httpHeaderLanguage,
		ContextKey:  contextLanguage,
		LoggerKey:   "lang",
		InfoLogOnce: true,
	})
}

func AddAuthentication(next http.Handler) http.Handler {
	return translateHTTPHeaderToContextValue(next, headerConfig{
		Name:       "authentication",
		HTTPHeader: httpHeaderAuthenticationToken,
		ContextKey: contextAuthenticationToken,
	})
}

func AddUserID(next http.Handler) http.Handler {
	return translateHTTPHeaderToContextValue(next, headerConfig{
		Name:       "user ID",
		HTTPHeader: httpHeaderUserID,
		ContextKey: contextUserID,
	})
}

func ctxGetStringValueOrEmptyString(ctx context.Context, value string) string {
	if lang, ok := ctx.Value(value).(string); ok {
		return lang
	}
	return ""
}
func CtxGetUserID(ctx context.Context) string {
	return ctxGetStringValueOrEmptyString(ctx, contextUserID)
}
func CtxGetAuthentication(ctx context.Context) string {
	return ctxGetStringValueOrEmptyString(ctx, contextAuthenticationToken)
}
func CtxGetLanguage(ctx context.Context) string {
	return ctxGetStringValueOrEmptyString(ctx, contextLanguage)
}

// testing purposes only
func TestingCtxNewWithLanguage(ctx context.Context, lang string) context.Context {
	return context.WithValue(ctx, contextLanguage, lang)
}

// testing purposes only
func TestingCtxNewWithAuthentication(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, contextAuthenticationToken, token)
}

type headerConfig struct {
	Name       string
	HTTPHeader string
	ContextKey string
	// if non-empty, the HTTPHeader content will be added to *every* log output
	// done from the request context
	LoggerKey   string
	InfoLogOnce bool
}

func translateHTTPHeaderToContextValue(next http.Handler, conf headerConfig) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if header, ok := r.Header[conf.HTTPHeader]; ok && len(header) == 1 {
			value := header[0]
			ctx = context.WithValue(ctx, conf.ContextKey, value)
			if conf.LoggerKey != "" {
				logger := log.With().Str(conf.LoggerKey, value).Logger()
				ctx = logger.WithContext(ctx)
			}
			r = r.WithContext(ctx)
		} else {
			log.Warn().Msgf("no %s HTTP header (key='%s') found in request: %v", conf.Name, conf.HTTPHeader, r.Header)
		}
		if conf.InfoLogOnce {
			log.Ctx(ctx).Info().Msgf("r=%v, headers=%v", r.RemoteAddr, r.Header)
		}
		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}
