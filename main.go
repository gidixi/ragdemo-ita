package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/ollama/ollama/api"
	"github.com/pgvector/pgvector-go"
)

// Funzione per inserire un documento e il suo embedding nel database
func insert(docfile string) error {
	// Legge il contenuto del file di documento
	docb, err := os.ReadFile(docfile)
	if err != nil {
		return err
	}
	doc := string(docb)

	// Crea un client API utilizzando le variabili d'ambiente
	cli, err := api.ClientFromEnvironment()
	if err != nil {
		return err
	}
	// Crea una richiesta di embedding per il documento
	req := &api.EmbeddingRequest{
		Model:  "llama3",
		Prompt: doc,
	}
	// Ottiene l'embedding del documento
	resp, err := cli.Embeddings(context.Background(), req)
	if err != nil {
		return err
	}

	// Converte l'embedding in un slice di float32
	e := make([]float32, len(resp.Embedding))
	for i, f := range resp.Embedding {
		e[i] = float32(f)
	}

	// Connette al database PostgreSQL
	conn, err := pgx.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		return err
	}
	defer conn.Close(context.Background())

	// Inserisce il documento e il suo embedding nella tabella 'items'
	_, err = conn.Exec(context.Background(),
		"INSERT INTO items (doc, embedding) VALUES ($1, $2)",
		doc, pgvector.NewVector(e))
	return err
}

// Funzione per eseguire una query utilizzando il documento più rilevante
func query(prompt string) (string, error) {
	// Crea un client API utilizzando le variabili d'ambiente
	cli, err := api.ClientFromEnvironment()
	if err != nil {
		return "", err
	}
	// Crea una richiesta di embedding per il prompt
	req := &api.EmbeddingRequest{
		Model:  "llama3",
		Prompt: prompt,
	}
	// Ottiene l'embedding del prompt
	resp, err := cli.Embeddings(context.Background(), req)
	if err != nil {
		return "", err
	}

	// Converte l'embedding in un slice di float32
	e := make([]float32, len(resp.Embedding))
	for i, f := range resp.Embedding {
		e[i] = float32(f)
	}

	// Connette al database PostgreSQL
	conn, err := pgx.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		return "", err
	}
	defer conn.Close(context.Background())

	// Seleziona il documento più simile al prompt basandosi sull'embedding
	var doc string
	err = conn.QueryRow(context.Background(),
		"SELECT doc FROM items ORDER BY embedding <-> $1 LIMIT 1",
		pgvector.NewVector(e)).
		Scan(&doc)
	if err != nil {
		return "", err
	}
	//fmt.Printf("** using document: %s\n", doc[:30])

	// Configura la richiesta di chat con il modello "llama3"
	stream := false
	req2 := &api.ChatRequest{
		Model:  "llama3",
		Stream: &stream,
		Messages: []api.Message{
			{
				Role: "user",
				Content: fmt.Sprintf(`Using the given reference text, succinctly answer the question that follows:
reference text is:

%s

end of reference text. The question is:

%s
`, doc, prompt),
			},
		},
	}
	// Funzione per gestire la risposta della chat
	var response string
	resp2fn := func(resp2 api.ChatResponse) error {
		response = strings.TrimSpace(resp2.Message.Content)
		return nil
	}
	// Esegue la richiesta di chat
	err = cli.Chat(context.Background(), req2, resp2fn)
	if err != nil {
		return "", err
	}
	return response, nil
}

// Definisce i flag per l'inserimento e la query
var (
	insertFlag = flag.Bool("insert", false, "insert a document and it's embedding")
	queryFlag  = flag.Bool("query", false, "answer a query using the most relevant document")
)

func usage() {
	fmt.Fprintf(flag.CommandLine.Output(), `Usage:
  %s -insert {path-to-doc-file}
  %s -query {query-text}

  Environment variables:
    DATABASE_URL  url of database, like postgres://host/dbname
    OLLAMA_HOST   url or host:port of OLLAMA server
    PG*           standard postgres env. vars are understood
`, os.Args[0], os.Args[0])
}

func main() {
	flag.Parse()
	if (!*insertFlag && !*queryFlag) || flag.NArg() != 1 || (*insertFlag && *queryFlag) {
		usage()
		os.Exit(1)
	}

	if *insertFlag {
		if err := insert(flag.Arg(0)); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	} else {
		if doc, err := query(flag.Arg(0)); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		} else {
			fmt.Println(doc)
		}
	}
}
