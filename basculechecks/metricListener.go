package basculechecks

import (
	"time"

	"github.com/Comcast/comcast-bascule/bascule"
	"github.com/Comcast/comcast-bascule/bascule/basculehttp"
	"github.com/SermoDigital/jose/jwt"
)

type MetricListener struct {
	expLeeway time.Duration
	nbfLeeway time.Duration
	measures  *JWTValidationMeasures
}

func (m *MetricListener) OnAuthenticated(auth bascule.Authentication) {
	now := time.Now()

	if m.measures == nil {
		return // measure tools are not defined, skip
	}

	if auth.Token == nil {
		return
	}

	c, ok := auth.Token.Attributes().Get("claims")
	if !ok {
		return // if there aren't any claims, skip
	}
	claims, ok := c.(jwt.Claims)
	if !ok {
		return // if claims aren't what we expect, skip
	}

	//how far did we land from the NBF (in seconds): ie. -1 means 1 sec before, 1 means 1 sec after
	if nbf, nbfPresent := claims.NotBefore(); nbfPresent {
		nbf = nbf.Add(-m.nbfLeeway)
		offsetToNBF := now.Sub(nbf).Seconds()
		m.measures.NBFHistogram.Observe(offsetToNBF)
	}

	//how far did we land from the EXP (in seconds): ie. -1 means 1 sec before, 1 means 1 sec after
	if exp, expPresent := claims.Expiration(); expPresent {
		exp = exp.Add(m.expLeeway)
		offsetToEXP := now.Sub(exp).Seconds()
		m.measures.ExpHistogram.Observe(offsetToEXP)
	}
}

func (m *MetricListener) OnErrorResponse(e basculehttp.ErrorResponseReason, _ error) {
	m.measures.ValidationReason.With(ReasonLabel, e.String()).Add(1)
}

type Option func(m *MetricListener)

func WithExpLeeway(e time.Duration) Option {
	return func(m *MetricListener) {
		m.expLeeway = e
	}
}

func WithNbfLeeway(n time.Duration) Option {
	return func(m *MetricListener) {
		m.nbfLeeway = n
	}
}

func NewMetricListener(m *JWTValidationMeasures, options ...Option) *MetricListener {
	listener := MetricListener{
		measures: m,
	}

	for _, o := range options {
		o(&listener)
	}
	return &listener
}
