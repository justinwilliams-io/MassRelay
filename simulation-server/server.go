package simulationserver

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

type Response struct {
	Response bool     `json:"Response"`
	Errors   []string `json:"Errors"`
}

func handleRequest(c echo.Context) error {
	sleepDuration := rand.Intn(1) + 1

	errorChance := rand.Intn(10)

	response := Response{
		Response: true,
		Errors:   []string{},
	}

	if errorChance == 0 {
		response.Errors = append(response.Errors, "Error!")
		fmt.Println("Simulated Error!")
	}

	time.Sleep(time.Duration(sleepDuration) * time.Second)

	return c.JSON(http.StatusOK, response)
}

func StartServer(c context.Context) {
	app := echo.New()

	app.POST("/", handleRequest)

	go func() {
		for {
			select {
			case <-c.Done():
				app.Shutdown(c)
			}
		}
	}()

    app.Start(":8080")
}
