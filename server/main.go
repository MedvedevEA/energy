package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	_ "github.com/lib/pq"
)

var mutexAddValue sync.Mutex

type AddValueReq struct {
	DeviceId *uuid.UUID `json:"device_id" binding:"required"`
	Value    int        `json:"value" binding:"required"`
}

func (a *Application) addValue(ctx *gin.Context) {
	req := new(AddValueReq)
	if err := ctx.ShouldBindJSON(req); err != nil {
		a.logger.Printf("addValue error: %s", err)
		ctx.Status(400)
	}
	_, err := a.db.Exec("INSERT INTO marks VALUES (gen_random_uuid(),$1,$2,now())", req.DeviceId, req.Value)
	if err != nil {
		a.logger.Printf("addValue error: %s", err)
		ctx.Status(500)
		return
	}
	ctx.Status(201)
}
func (a *Application) addValueTurn(ctx *gin.Context) {
	req := new(AddValueReq)
	if err := ctx.ShouldBindJSON(req); err != nil {
		a.logger.Printf("addValueTurn error: %s", err)
		ctx.Status(400)
	}
	mutexAddValue.Lock()
	_, err := a.db.Exec("INSERT INTO marks VALUES (gen_random_uuid(),$1,$2,now())", req.DeviceId, req.Value)
	mutexAddValue.Unlock()
	if err != nil {
		a.logger.Printf("addValueTurn error: %s", err)
		ctx.Status(500)
		return
	}
	ctx.Status(201)
}

type Application struct {
	logger *log.Logger
	db     *sql.DB
}

func (app *Application) routes() *gin.Engine {
	router := gin.Default()
	router.POST("/v1/values", app.addValue)
	router.POST("/v2/values", app.addValueTurn)
	return router
}
func main() {

	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime)

	//Подключение базы даннных
	db, err := sql.Open("postgres", os.Getenv("DATASOURCENAME"))
	if err != nil {
		logger.Fatal(err)
	}
	defer db.Close()
	if _, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS marks (
		mark_id uuid NOT NULL DEFAULT gen_random_uuid(),
		device_id uuid NOT NULL,
		value integer NOT NULL,
		time_stamp timestamp with time zone NOT NULL,
		CONSTRAINT values_pkey PRIMARY KEY (mark_id) );
	`); err != nil {
		logger.Fatal(err)
	}

	app := &Application{
		db:     db,
		logger: logger,
	}

	srv := &http.Server{
		Addr:         ":8000",
		Handler:      app.routes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	logger.Printf("API Server 'Energy' is started in addr:[8000]")
	if err := srv.ListenAndServe(); err != nil {
		logger.Printf("API Server 'Energy' error: %s", err)
	}

}
