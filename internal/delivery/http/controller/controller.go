package controller

import "github.com/faruqrahmadani/slack-thread-alert/internal/usecase"

type Controller struct {
	DatadogController *DatadogController
	ScalyrController  *ScalyrController
}

func NewController(alertUC usecase.AlertUseCaseItf) *Controller {
	return &Controller{
		DatadogController: NewDatadogController(alertUC),
		ScalyrController:  NewScalyrController(alertUC),
	}
}
