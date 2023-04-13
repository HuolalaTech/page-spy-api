package middleware

import (
	"strconv"
	"time"

	"github.com/HuolalaTech/page-spy-api/logger"
	"github.com/HuolalaTech/page-spy-api/metric"
	"github.com/google/uuid"
	echo "github.com/labstack/echo/v4"
)

const HeaderXRequestID = "X-Request-ID"

var middlewareLogger = logger.Log().WithField("_module", "middleware")

func Logger() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			startTime := time.Now()
			rid := c.Request().Header.Get(HeaderXRequestID)
			if rid == "" {
				rid = uuid.New().String()
				c.Response().Header().Set(HeaderXRequestID, rid)
			}

			logger := middlewareLogger.WithField("_request_id", rid)
			route := "/404"
			for _, r := range c.Echo().Routes() {
				if r.Method == c.Request().Method && r.Path == c.Path() {
					route = r.Name
				}
			}
			err := next(c)
			endTime := time.Now()
			latencyTime := endTime.Sub(startTime)
			reqMethod := c.Request().Method
			reqURI := c.Request().RequestURI
			statusCode := c.Response().Status
			clientIP := c.RealIP()
			metric.Time("ap_request", map[string]string{
				"method": reqMethod,
				"route":  route,
				"ret":    strconv.Itoa(statusCode),
			}, float64(time.Since(startTime)))

			l := logger.Infof
			errString := "-"
			if statusCode >= 500 {
				l = logger.Errorf
			}

			if err != nil {
				errString = err.Error()
			}

			l("| %3d | %13v | %15s | %s | %s | %s",
				statusCode,
				latencyTime,
				clientIP,
				reqMethod,
				reqURI,
				errString,
			)
			return err
		}
	}
}
