package http

import (
	"net/http"

	"github.com/faruqrahmadani/slack-thread-alert/internal/delivery/http/controller"
	"github.com/labstack/echo/v4"
)

func registerRouters(e *echo.Echo, ctrl *controller.Controller) {
	e.GET("/health", healthcheckHandler)

	webhook := e.Group("/webhook")
	webhook.POST("/datadog", ctrl.DatadogController.DatadogAlertWebhook)
}

func healthcheckHandler(c echo.Context) error {
	return c.JSON(http.StatusOK, echo.Map{"message": "ok"})
}
