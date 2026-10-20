package controller

import (
	"errors"
	"net/http"

	"github.com/faruqrahmadani/slack-thread-alert/internal/domain"
	"github.com/faruqrahmadani/slack-thread-alert/internal/usecase"
	"github.com/labstack/echo/v4"
)

type DatadogController struct {
	alertUC usecase.AlertUseCaseItf
}

func NewDatadogController(alertUC usecase.AlertUseCaseItf) *DatadogController {
	return &DatadogController{alertUC: alertUC}
}

// DatadogAlertWebhook handles POST /webhook/datadog
func (ctrl *DatadogController) DatadogAlertWebhook(c echo.Context) error {
	var req domain.DatadogAlertWebhookRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "invalid payload"})
	}

	alert := req.ToAlert()
	if err := ctrl.alertUC.Handle(c.Request().Context(), &alert); err != nil {
		// 4xx is not retried by Datadog, 5xx is
		if errors.Is(err, domain.ErrInvalidAlert) {
			return c.JSON(http.StatusBadRequest, echo.Map{"message": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{"message": err.Error()})
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "ok"})
}
