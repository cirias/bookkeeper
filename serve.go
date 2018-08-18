package main

import (
	"fmt"
	"log"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/cirias/tgbot"
	"github.com/pkg/errors"
)

func serve(bot *tgbot.Bot, sheet *Sheet) error {
	params := &tgbot.GetUpdatesParams{
		Offset:  0,
		Limit:   10,
		Timeout: 10,
	}

	for {
		var updates []*tgbot.Update
		err := willRetry(func() error {
			var err error
			updates, err = bot.GetUpdates(params)
			return errors.Wrap(err, "could not get updates")
		}, 4)
		if err != nil {
			return err
		}

		for _, u := range updates {
			go func() {
				reply := handleUpdate(sheet, u)

				if reply == "" {
					return
				}

				if err := sendMessage(bot, u.Message.Chat.Id, reply); err != nil {
					log.Println(err)
				}
			}()
		}

		if len(updates) > 0 {
			params.Offset = updates[len(updates)-1].Id + 1
		}
	}
}

func handleUpdate(sheet *Sheet, u *tgbot.Update) string {
	payment, err := parseMessage(u.Message)
	if err != nil {
		return err.Error()
	}

	if err := sheet.Append(payment.Values()); err != nil {
		return err.Error()
	}

	return fmt.Sprintf("roger: %s", payment.String())
}

var kRecordReg = regexp.MustCompile(`([^\d]{2,})(\d+(?:\.\d*)?)\s*$`)
var kUsers = map[int64]string{
	119838553: "Sirius",
	500028413: "Jian",
}

// China doesn't have daylight saving. It uses a fixed 8 hour offset from UTC.
var kBeijing = time.FixedZone("Beijing Time", int((8 * time.Hour).Seconds()))
var kTimeFormat = "2006-01-02 15:04:05"

func parseMessage(m *tgbot.Message) (*Payment, error) {
	user, ok := kUsers[m.From.Id]
	if !ok {
		return nil, errors.Errorf("unknown user %d\n", m.From.Id)
	}

	matches := kRecordReg.FindStringSubmatch(m.Text)
	if len(matches) < 3 {
		return nil, errors.Errorf("invalid message: %s", m.Text)
	}

	name := strings.Trim(matches[1], " ")
	money, err := strconv.ParseFloat(matches[2], 64)
	if err != nil {
		return nil, errors.Errorf("could not convert %s to float", matches[2])
	}

	timestamp := time.Now()

	return &Payment{
		name:      name,
		money:     money,
		user:      user,
		timestamp: timestamp,
	}, nil
}

type Payment struct {
	name      string
	money     float64
	user      string
	category  string
	timestamp time.Time
}

func (p *Payment) String() string {
	return fmt.Sprintf("%s spent %.2f on %s", p.user, p.money, p.name)
}

func (p *Payment) Values() []interface{} {
	return []interface{}{
		p.name,
		p.money,
		p.user,
		p.category,
		p.timestamp.In(kBeijing).Format(kTimeFormat),
	}
}

func sendMessage(bot *tgbot.Bot, chatId int64, text string) error {
	return willRetry(func() error {
		_, err := bot.SendMessage(&tgbot.SendMessageParams{
			ChatId:    chatId,
			Text:      text,
			ParseMode: "markdown",
		})
		return errors.Wrap(err, "could not send message with tgbot")
	}, 4)
}

func willRetry(fn func() error, n int) error {
	err := fn()
	for i := 0; i < n; i += 1 {
		if err == nil {
			break
		}
		time.Sleep(time.Duration(math.Pow10(i)) * time.Millisecond * 100) // 0.1, 1, 10, 100

		log.Printf("retry %d: %v\n", i, err)
		err = fn()
	}

	return err
}
