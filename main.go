package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"

	"context"
	"fmt"

	"go.opentelemetry.io/otel"

	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)


type Men struct {
	Name string `json:"name"`
	Country string `json:"country"`
}

func initTracer() {
	ctx := context.Background()

	client := otlptracehttp.NewClient()
	exporter, err := otlptrace.New(ctx, client)
	if err != nil {
		log.Fatalf("failed to initialize exporter: %e", err)
	}

	res, err := resource.New(ctx)
	if err != nil {
		log.Fatalf("failed to initialize resource: %e", err)
	}

	// Create the trace provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	// Set the global trace provider
	otel.SetTracerProvider(tp)

	// Set the propagator
	propagator := propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{})
	otel.SetTextMapPropagator(propagator)
}

func helloHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	fmt.Fprintf(w, "Hello, %s!", vars["name"])
}

func main() {
	initTracer()

	//connection to database
	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
    if err!= nil {
        log.Fatal(err)
    }
    defer db.Close()

    //create table if it doesn't exist
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS men( name text, country text);")
    if err!= nil {
        log.Fatal(err)
    }

    //create router
	router := mux.NewRouter()
	router.Use(otelmux.Middleware("my-server"))

	//router.Use(nrgorilla.Middleware(app))
	router.HandleFunc("/men", getMen(db)).Methods("GET")
	router.HandleFunc("/men/{id}", getMenByID(db)).Methods("GET")
	router.HandleFunc("/men", createUser(db)).Methods("POST")
	router.HandleFunc("/men/{id}", updateUser(db)).Methods("PUT")
	router.HandleFunc("/men/{id}", deleteUser(db)).Methods("DELETE")

	//start server
    log.Fatal(http.ListenAndServe(":8000", jsonContentTypeMiddleware(router)))

}

func jsonContentTypeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        next.ServeHTTP(w, r)
    })
}

//get all mens
func getMen(db *sql.DB) func(w http.ResponseWriter, r *http.Request) {
    return func(w http.ResponseWriter, r *http.Request) {
        rows, err := db.Query("SELECT name, country FROM men")
        if err!= nil {
            log.Fatal(err)
        }
        defer rows.Close()

        var men Men
        for rows.Next() {
            err := rows.Scan( &men.Name, &men.Country)
            if err!= nil {
                log.Fatal(err)
            }
        }

        if err := rows.Err(); err!= nil {
            log.Fatal(err)
        }

        json.NewEncoder(w).Encode(men)
    }
}

//get user by id
func getMenByID(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]
		row := db.QueryRow("SELECT  name, country FROM men WHERE id = $1", id)
		var men Men
		err := row.Scan( &men.Name, &men.Country)
		if err!= nil {
			log.Fatal(err)
		}

		json.NewEncoder(w).Encode(men)
	}
}

//create user
func createUser(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
			var men Men
			err := json.NewDecoder(r.Body).Decode(&men)
			if err!= nil {
				log.Fatal(err)
			}
	
			_, err = db.Exec("INSERT INTO men (name, country) VALUES ($1, $2)", men.Name, men.Country)
			if err!= nil {
				log.Fatal(err)
			}
	
			json.NewEncoder(w).Encode(men)
		}
}

//update user
func updateUser(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        id := mux.Vars(r)["id"]
        var men Men
        err := json.NewDecoder(r.Body).Decode(&men)
        if err!= nil {
            log.Fatal(err)
        }

        _, err = db.Exec("UPDATE men SET name = $1, country = $2 WHERE id = $3", men.Name, men.Country, id)
        if err!= nil {
            log.Fatal(err)
        }

        json.NewEncoder(w).Encode(men)
    }
}

//delete user
func deleteUser(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]
		_, err := db.Exec("DELETE FROM men WHERE id = $1", id)
		if err!= nil {
			log.Fatal(err)
		}

		json.NewEncoder(w).Encode(id)
	}
}