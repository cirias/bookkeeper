package main

import (
	"flag"
	"io/ioutil"
	"log"

	"github.com/cirias/tgbot"
)

var (
	botToken            = flag.String("token", "", "telegram bot token")
	credentialsFilepath = flag.String("credentials", "credentials.json", "filepath of google credentials")
	spreadsheetId       = flag.String("sheet", "", "id of google sheet")
)

func main() {
	flag.Parse()

	credentials, err := ioutil.ReadFile(*credentialsFilepath)
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	sheet, err := NewSheet(credentials, *spreadsheetId)
	if err != nil {
		log.Fatalln(err)
	}

	bot := tgbot.NewBot(*botToken)
	if err := serve(bot, sheet); err != nil {
		log.Fatalln(err)
	}
}
