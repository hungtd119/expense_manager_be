package main

import (
	"log"

	"expense-manager-mvp/internal/adapter/httpapi"
	"expense-manager-mvp/internal/app"
)

func main() {
	if err := app.Run(httpapi.NewHandler); err != nil {
		log.Fatal(err)
	}
}
