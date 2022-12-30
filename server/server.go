package server

import (
	"dcns/model"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"net/http"
)

var config *model.Config
var updateChannel chan *model.DockerUpdateContext

type PushUpdateRequest struct {
	Status   string `json:"status"`
	Image    string `json:"image"`
	Created  string `json:"created"`
	HubLink  string `json:"hub_link"`
	Digest   string `json:"digest"`
	Hostname string `json:"hostname"`
}

type PushUpdateResponse struct {
	Success      bool   `json:"success"`
	ErrorMessage string `json:"errorMessage,omitempty"`
}

func StartServer(uC chan *model.DockerUpdateContext, c *model.Config) {
	config = c
	updateChannel = uC

	// Echo instance
	e := echo.New()
	e.HideBanner = true

	// Middleware
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "${method} | ${uri} => ${status}\n",
	}))
	e.Use(middleware.Recover())

	// Routes
	e.POST("/update/push", handleUpdatePush)

	// Start server
	e.Logger.Fatal(e.Start(config.Host + ":" + config.Port))
}

func handleUpdatePush(c echo.Context) error {
	var request PushUpdateRequest
	err := c.Bind(&request)

	if err != nil {
		return c.JSON(http.StatusBadRequest, &PushUpdateResponse{
			Success:      false,
			ErrorMessage: "Invalid JSON request",
		})
	}

	if request.Image == "" || request.Status == "" || request.Created == "" {
		return c.JSON(http.StatusBadRequest, &PushUpdateResponse{
			Success:      false,
			ErrorMessage: "Not all values are filled",
		})
	}

	for _, c := range *config.ContainerContexts {
		if c.Image == request.Image {
			// Find
			updateChannel <- &model.DockerUpdateContext{ContainerContext: &c, Created: request.Created, HubLink: request.HubLink, Digest: request.Digest, Hostname: request.Hostname}
		}
	}

	return c.JSON(http.StatusOK, &PushUpdateResponse{
		Success: true,
	})
}
