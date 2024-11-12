package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	_ "modernc.org/sqlite"
)

type Cotacao struct {
	Usdbrl struct {
		Code       string `json:"code"`
		Codein     string `json:"codein"`
		Name       string `json:"name"`
		High       string `json:"high"`
		Low        string `json:"low"`
		VarBid     string `json:"varBid"`
		PctChange  string `json:"pctChange"`
		Bid        string `json:"bid"`
		Ask        string `json:"ask"`
		Timestamp  string `json:"timestamp"`
		CreateDate string `json:"create_date"`
	} `json:"USDBRL"`
}

var DB *sql.DB

func main() {
	DB = initDB("cotacao.db")
	defer DB.Close()
	http.HandleFunc("/cotacao", BuscaCotacaoHandler)
	http.ListenAndServe(":8080", nil)
}

func BuscaCotacaoHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/cotacao" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	cotacao, error := BuscaCotacao()
	if error != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Contet-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	result, error := json.Marshal(cotacao.Usdbrl.Bid)
	if error != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Write(result)

}

func BuscaCotacao() (*Cotacao, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	select {
	case <-ctx.Done():
		log.Println("timeout when processing the request.")
	default:
		log.Println("processing...")
	}

	request, err := http.NewRequestWithContext(ctx, "GET", "https://economia.awesomeapi.com.br/json/last/USD-BRL", nil)
	if err != nil {
		return nil, err
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	var cotacao Cotacao
	err = json.Unmarshal(body, &cotacao)
	if err != nil {
		return nil, err
	}

	err = CotacaoWriteDB(&cotacao)
	if err != nil {
		return nil, err
	}

	return &cotacao, nil
}

func CotacaoWriteDB(cotacao *Cotacao) error {
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	select {
	case <-ctx.Done():
		log.Println("timeout when writing to database.")
	default:
		log.Println("processing...")
	}

	jsonData, err := json.Marshal(cotacao)
	if err != nil {
		return err
	}

	err = insertCotacao(DB, jsonData)
	if err != nil {
		return err
	}

	return nil
}

func initDB(filepath string) *sql.DB {
	db, err := sql.Open("sqlite", filepath)
	if err != nil {
		log.Println(err)
	} else if db == nil {
		log.Println("db is nil")
	}

	createTable(db)
	return db
}

func createTable(db *sql.DB) error {
	sqlTable := `CREATE TABLE IF NOT EXISTS quotation(
	id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
	json TEXT NOT NULL);`

	_, err := db.Exec(sqlTable)

	if err != nil {
		log.Printf("Failed to create table: %v", err)
		return err
	}
	return nil
}

func insertCotacao(db *sql.DB, jsonData []byte) error {
	_, err := db.Exec("INSERT INTO quotation (json) VALUES (?)", jsonData)
	if err != nil {
		log.Printf("Failed to insert registry: %v", err)
		return err
	}
	return nil
}
