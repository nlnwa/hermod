package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/nlnwa/heimdall/pdp"
	"net/http"
	"net/url"
)

func main() {

	e := echo.New()
	e.Use(middleware.Recover())
	e.Use(middleware.Logger())

	playbackIndexUrl, err := url.Parse("http://localhost:8085/")

	if err != nil {
		e.Logger.Fatal(err)
	}

	targets := []*middleware.ProxyTarget{
		{
			URL: playbackIndexUrl,
		},
	}

	// g := e.Group("/web")
	e.Use(middlewarePdp)
	e.Use(middleware.Proxy(middleware.NewRoundRobinBalancer(targets)))

	e.Logger.Fatal(e.Start(":1323"))
}

func middlewarePdp(next echo.HandlerFunc) echo.HandlerFunc {
	pdpUrl := "http://localhost:8087/auth"
	accRes := pdp.AccessResponse{}

	return func(c echo.Context) error {
		fmt.Println("from middlewarePdp")

		accRec := pdp.AccessRequest{
			Url:   c.QueryParam("url"),
			Token: c.QueryParam("token"),
		}

		fmt.Printf("Generating AccessRequest %#+v", accRec)

		b, err := json.Marshal(accRec)
		if err != nil {
			return fmt.Errorf("failed reading the request body %s", err)
		}
		body := bytes.NewReader(b)

		req, err := http.NewRequest("POST", pdpUrl, body)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

		if err != nil {
			return fmt.Errorf("failed creating request %s", err)
		}

		client := &http.Client{}

		response, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("failed sending request %s", err)
		}

		defer response.Body.Close()

		err = json.NewDecoder(response.Body).Decode(&accRes)
		if err != nil {
			return fmt.Errorf("failed decoding response %s", err)
		}

		if accRes.Permission == pdp.Deny {
			return echo.NewHTTPError(http.StatusForbidden, "Access denied")
		} else if accRes.Permission == pdp.Allow {
			fmt.Println("...Access granted")
			return next(c)
		}
		return next(c)
	}
}
