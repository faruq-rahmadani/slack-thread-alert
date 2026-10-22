package controller

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/faruqrahmadani/slack-thread-alert/internal/domain"
	"github.com/faruqrahmadani/slack-thread-alert/internal/usecase"
	"github.com/labstack/echo/v4"
)

type ScalyrController struct {
	alertUC usecase.AlertUseCaseItf
}

func NewScalyrController(alertUC usecase.AlertUseCaseItf) *ScalyrController {
	return &ScalyrController{alertUC: alertUC}
}

// ScalyrAlertWebhook handles POST /webhook/scalyr?channel_id=<slack channel id>
func (ctrl *ScalyrController) ScalyrAlertWebhook(c echo.Context) error {
	// Scalyr sends a JSON body with content-type application/x-www-form-urlencoded
	// unless the webhook sets &content-type=application/json, so decode it as JSON regardless
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "invalid payload"})
	}

	var req domain.ScalyrAlertWebhookRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "invalid payload"})
	}
	if req.ChannelID == "" {
		req.ChannelID = c.QueryParam("channel_id")
	}

	alert := req.ToAlert()
	if err := ctrl.alertUC.Handle(c.Request().Context(), &alert); err != nil {
		if errors.Is(err, domain.ErrInvalidAlert) {
			return c.JSON(http.StatusBadRequest, echo.Map{"message": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{"message": err.Error()})
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "ok"})
}
