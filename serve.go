package main

import (
	"fmt"
	"log"
	"math"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cirias/tgbot"
	"github.com/pkg/errors"
)

var kRecordReg = regexp.MustCompile(`([^\d]{2,})(\d+(?:\.\d*)?)\s*$`)

// China doesn't have daylight saving. It uses a fixed 8 hour offset from UTC.
var kBeijing = time.FixedZone("Beijing Time", int((8 * time.Hour).Seconds()))
var kTimeFormat = "2006-01-02 15:04:05"

type Server struct {
	bot   *tgbot.Bot
	sheet *Sheet
	users map[int64]string
	admin int64
	chats *sync.Map
}

func NewServer(bot *tgbot.Bot, sheet *Sheet, users, admin string) (s *Server, err error) {
	s = &Server{
		bot:   bot,
		sheet: sheet,
		chats: &sync.Map{},
	}

	s.users, err = parseUsers(users)
	if err != nil {
		return nil, err
	}

	for id, name := range s.users {
		if admin == name {
			s.admin = id
		}
	}
	if s.admin == 0 {
		return nil, errors.Errorf("could not found admin in users")
	}

	return s, nil
}

func parseUsers(users string) (map[int64]string, error) {
	m := make(map[int64]string)

	pairs := strings.Split(users, ",")
	for _, pair := range pairs {
		vs := strings.Split(pair, "=")
		if len(vs) != 2 {
			return nil, errors.Errorf("invalid user: %s", pair)
		}

		id, err := strconv.ParseInt(vs[1], 10, 64)
		if err != nil {
			return nil, errors.Errorf("invalid user: %s", pair)
		}

		m[id] = vs[0]
	}

	return m, nil
}

func (s *Server) serve() error {
	params := &tgbot.GetUpdatesParams{
		Offset:  0,
		Limit:   10,
		Timeout: 10,
	}

	for {
		var updates []*tgbot.Update
		err := willRetry(func() error {
			var err error
			updates, err = s.bot.GetUpdates(params)
			return errors.Wrap(err, "could not get updates")
		}, 4)
		if err != nil {
			return err
		}

		for _, u := range updates {
			go s.handleUpdate(u)
		}

		if len(updates) > 0 {
			params.Offset = updates[len(updates)-1].Id + 1
		}
	}
}

func (s *Server) handleUpdate(u *tgbot.Update) {
	s.chats.LoadOrStore(u.Message.From.Id, u.Message.Chat.Id)

	payment, err := s.parseMessage(u.Message)
	if err != nil {
		s.handleUpdateError(err, "could not parse message")
		return
	}

	if err := s.sheet.Append(payment.Values()); err != nil {
		s.handleUpdateError(err, "could not append to sheet")
		return
	}

	reply := fmt.Sprintf("roger: %s", payment)
	go mustSendMessage(s.bot, u.Message.Chat.Id, reply)

	notification := fmt.Sprintf("note: %s", payment)
	for uid := range s.users {
		if uid == u.Message.From.Id {
			continue
		}

		chatId, _ := s.chats.Load(uid)
		if chatId == nil {
			continue
		}

		go mustSendMessage(s.bot, chatId.(int64), notification)
	}
}

func (s *Server) handleUpdateError(err error, msg string) {
	errContent := fmt.Sprintf("%s: %s", msg, err)
	log.Println(errContent)

	chatId, _ := s.chats.Load(s.admin)
	if chatId == nil {
		log.Printf("could not load chat id of admin user: %d\n", s.admin)
		return
	}

	go mustSendMessage(s.bot, chatId.(int64), errContent)
}

func (s *Server) parseMessage(m *tgbot.Message) (*Payment, error) {
	user, ok := s.users[m.From.Id]
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

func mustSendMessage(bot *tgbot.Bot, chatId int64, text string) {
	if err := sendMessage(bot, chatId, text); err != nil {
		log.Println(err)
	}
}

func sendMessage(bot *tgbot.Bot, chatId int64, text string) error {
	return willRetry(func() error {
		_, err := bot.SendMessage(&tgbot.SendMessageParams{
			ChatId: chatId,
			Text:   text,
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
