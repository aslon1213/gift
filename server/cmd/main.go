package main

import (
	"aslon1213/gift/pkg/app"
	"log"
)

// @title           Gift API
// @version         0.1.0
// @description     This is a Gift API server.
// @termsOfService  https://github.com/aslon1213/gift

// @contact.name   API Support
// @contact.url    https://github.com/aslon1213/gift
// @contact.email  hamidovaslon1@gmail.com

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:3000
// @BasePath  /api/v1

// @securityDefinitions.basic  BasicAuth

// @externalDocs.description  OpenAPI
// @externalDocs.url          https://swagger.io/resources/open-api/
func main() {
	app := app.NewApp()
	log.Fatal(app.Start())
}
