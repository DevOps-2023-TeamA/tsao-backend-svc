package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

type Accounts struct {
	ID       		int    	`json:"ID"`
	Name     		string 	`json:"Name"`
	Username     	string 	`json:"Username"`
	Password		string 	`json:"Password"`
	Role			string 	`json:"Role"`
	CreationDate	string	`json:"CreationDate"`
	IsApproved		bool 	`json:"IsApproved"`
	IsDeleted		bool 	`json:"IsDeleted"`
}

var connectionString string

func main() {
	r := mux.NewRouter()
    api := r.PathPrefix("/api/accounts").Subrouter()
    api.HandleFunc("", CreateAccount).Methods("POST")
    api.HandleFunc("", ReadAccounts).Methods("GET")
    api.HandleFunc("/{id}", UpdateAccount).Methods("PUT")
    api.HandleFunc("/{id}", DeleteAccount).Methods("DELETE")
    http.Handle("/", r)

	fmt.Println("Accounts microservice running on http://localhost:8002/api/accounts")
	
	// CORS configuration
    corsHandler := cors.New(cors.Options{
		AllowedOrigins: []string{"http://127.0.0.1:5502"}, // Your frontend origin
        AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
        AllowedHeaders: []string{"Content-Type"},
    })
	
	cmd := flag.String("sql", "", "")
	flag.Parse()
	connectionString = string(*cmd)

    handler := corsHandler.Handler(r)
	http.ListenAndServe(":8002", handler)
}

func CreateAccount(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-type", "application/json")
	log.Println("Entering endpoint to add new account")

	var newAccount Accounts
	err := json.NewDecoder(r.Body).Decode(&newAccount)
	if err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}
	sha := sha256.New()
	sha.Write([]byte(newAccount.Password))
	newAccount.Password = hex.EncodeToString(sha.Sum(nil))

	location, _ := time.LoadLocation("Asia/Singapore")
	newAccount.CreationDate = time.Now().In(location).Format("2006-01-02 15:04:05")

	newAccount.IsApproved = false
	newAccount.IsDeleted = false

	db, _ := sql.Open("mysql", connectionString)
	defer db.Close()

	existingUsername, err := checkInfo(db, newAccount.Username)
    if err != nil {
        log.Println(err)
        http.Error(w, "Error checking existing user", http.StatusInternalServerError)
        return
    }

    if existingUsername != nil {
        http.Error(w, "Username already exists", http.StatusConflict)
        return
    }

	result, err := db.Exec(
		`INSERT INTO tsao_accounts (Name, Username, Password, Role, CreationDate, IsApproved, IsDeleted)
		 VALUES (?, ?, ?, ?, ?, ?, ?);`,
		newAccount.Name, newAccount.Username, newAccount.Password, newAccount.Role,
		newAccount.CreationDate, newAccount.IsApproved, newAccount.IsDeleted)
	if err == nil  {
		accountID, _ := result.LastInsertId()
		newAccount.ID = int(accountID)
		
		newAccountJson, _ := json.Marshal(newAccount)
		if err != nil {
			log.Println(err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusAccepted)
		w.Write(newAccountJson)
	} else {
		log.Println(err)
		http.Error(w, "Error creating new account", http.StatusInternalServerError)
		return
	}
}

func ReadAccounts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-type", "application/json")
	log.Println("Entering endpoint to query all accounts")
}

func UpdateAccount(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-type", "application/json")
	log.Println("Entering endpoint to update an account's information")
}

func DeleteAccount(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-type", "application/json")
	log.Println("Entering endpoint to (soft) delete an account")
}

func checkInfo(db *sql.DB, username string) (*string, error) {
    var retrievedUsername string
    err := db.QueryRow(`
        SELECT Username FROM tsao_accounts
        WHERE Username=?;`, username).Scan(
        &retrievedUsername)

    if err != nil {
        if err == sql.ErrNoRows {
            return nil, nil // No existing user found
        }
        return nil, err
    }
    return &retrievedUsername, nil
}