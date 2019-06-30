package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	sheets "google.golang.org/api/sheets/v4"
)

type Sheet struct {
	spreadsheetId string
	service       *sheets.Service
}

func NewSheet(credentials []byte, spreadsheetId string) (*Sheet, error) {
	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(credentials, "https://www.googleapis.com/auth/spreadsheets")
	if err != nil {
		return nil, errors.Wrap(err, "could not parse client secret file to config")
	}
	client := getClient(config)

	service, err := sheets.New(client)
	if err != nil {
		return nil, errors.Wrap(err, "could not retrieve sheets client")
	}

	return &Sheet{
		spreadsheetId: spreadsheetId,
		service:       service,
	}, nil
}

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) *http.Client {
	tokFile := "token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}
	go autoRefreshTokenFile(config, tok, tokFile)
	return config.Client(context.Background(), tok)
}

func autoRefreshTokenFile(config *oauth2.Config, token *oauth2.Token, tokFile string) {
	for {
		time.Sleep(time.Hour)
		log.Println("refreshing token")
		tokenSource := config.TokenSource(context.Background(), token)
		t, err := tokenSource.Token()
		if err != nil {
			log.Println("could not refresh token:", err)
			continue
		}

		saveToken(tokFile, t)
	}
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(oauth2.NoContext, authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	defer f.Close()
	if err != nil {
		return nil, err
	}
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	defer f.Close()
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	json.NewEncoder(f).Encode(token)
}

func (s *Sheet) Append(values []interface{}) error {
	tableRange := "book"
	vr := &sheets.ValueRange{
		// Values: [][]interface{}{[]interface{}{"打车", 23.84, "Sirius", "", time.Now().Format(time.RFC3339)}},
		Values: [][]interface{}{values},
	}
	_, err := s.service.Spreadsheets.Values.Append(s.spreadsheetId, tableRange, vr).ValueInputOption("RAW").InsertDataOption("INSERT_ROWS").Do()
	return errors.Wrap(err, "could not append to sheet")
}
