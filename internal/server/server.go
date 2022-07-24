package server

import (
	"log"
	"net/http"
	"regexp"

	"github.com/PuerkitoBio/goquery"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
	store "github.com/vipowerus/reminder/internal/store"
)

// Config ...
type Config struct {
	LogLevel string `toml:"log_level"`
	Store    *store.Config
	BotToken string `toml:"bot_token"`
}

// NewConfig ...
func NewConfig() *Config {
	return &Config{
		LogLevel: "debug",
		Store:    store.NewConfig(),
	}
}

// APIServer ...
type Server struct {
	config      *Config
	logger      *logrus.Logger
	store       *store.Store
	bot         *tgbotapi.BotAPI
	updatesConf tgbotapi.UpdateConfig
}

// New ...
func New(config *Config) *Server {
	return &Server{
		config: config,
		logger: logrus.New(),
	}
}

// Start ...
func (s *Server) Start() error {
	if err := s.configureLogger(); err != nil {
		return err
	}

	if err := s.configureStore(); err != nil {
		return err
	}

	if err := s.configureBot(); err != nil {
		return err
	}

	s.configureBotUpdates()
	s.logger.Info("api server is started")

	s.handleBotUpdates()

	return nil
}

// handleBotUpdates ...
func (s *Server) handleBotUpdates() {
	updates := s.bot.GetUpdatesChan(s.updatesConf)
	for update := range updates {
		if update.Message != nil {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")

			switch update.Message.Command() {
			case "start":
				msg.Text += "Введите номер группы:"
				if err := s.store.AddUser(update.Message.From.ID); err != nil {
					s.logger.Error(err)
				}
				s.bot.Send(msg)
				continue
			case "help":
				msg.Text += "Здесь будет help когда-нибудь"
			case "change":
				if len(update.Message.CommandArguments()) == 0 {
					msg.Text = "Пожалуйста, добавьте к команде номер новой группы"
					s.bot.Send(msg)
					continue
				}
				s.logger.Println(update.Message.CommandArguments() + "!!!!!!!!!!")
				// change group here
			}

			switch update.Message.Text {
			case "open":
				msg.Text = "chid a"
			default:
				hasGroup, err := s.store.UserInGroup(update.Message.From.ID)
				if err != nil {
					s.logger.Error(err)
				}
				if hasGroup {
					msg.Text = "У вас уже есть группа. Вы можете сменить ее с помощью команды '/change *****'"
					s.bot.Send(msg)
					continue
				}
				if isGroupNumber(s, update.Message.Text) {
					resp, err := http.Get("https://table.nsu.ru/group/" + update.Message.Text)
					if err != nil || resp.StatusCode == 404 {
						msg.Text = "Не могу найти ваше расписание"
						s.bot.Send(msg)
						continue
					}
					exists, err := s.store.ScheduleExists(update.Message.Text)
					if err != nil {
						msg.Text = "Я сламался" // rework and add variable to Text!!!
						s.bot.Send(msg)
						s.logger.Error(err)
						continue
					}
					if !exists {
						schedule, err := Parse(resp, "div.subject", "div.room a")
						if err != nil {
							msg.Text = "Я сламался" // rework
							s.bot.Send(msg)
							s.logger.Error(err)
							continue
						}
						if err := s.store.AddSchedule(update.Message.Text, schedule); err != nil {
							msg.Text = "Я сламался" // rework
							s.bot.Send(msg)
							s.logger.Error(err)
							continue
						}
					}
					if err := s.store.AddUserToSchedule(update.Message.From.ID, update.Message.Text); err != nil {
						msg.Text = "Я сламался" // rework
						s.bot.Send(msg)
						s.logger.Error(err)
						continue
					}
					if err := s.store.UpdateUserHasGroup(true, update.Message.From.ID); err != nil { // CHECK IT!!!!!!!!!!
						msg.Text = "Я сламался" // rework
						s.bot.Send(msg)
						s.logger.Error(err)
						continue
					}
					msg.Text = "Вы добавлены на рассылку по расписанию группы " + update.Message.Text
					s.bot.Send(msg)
					continue

				} else {
					msg.Text = "Некорректный номер группы. Я умею работать с такими номерами: [0-9][0-9][0-9][0-9][0-9]"
					s.bot.Send(msg)
					continue
				}
			}

			s.bot.Send(msg)
		}
	}
}

func Parse(resp *http.Response, selector1, selector2 string) ([7][6]string, error) {
	var rooms [7][6]string
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return rooms, err
	}

	doc.Find("table.time-table td").Each(func(i int, s *goquery.Selection) {
		if i%7 == 0 {
			return
		}
		title, _ := s.Find("div.subject").Attr("title")
		room := s.Find("div.room a").First().Text()
		rooms[i/7][(i-1)%7] = title + " " + room
	})

	return rooms, nil
}

func (s *Server) configureLogger() error {
	level, err := logrus.ParseLevel(s.config.LogLevel)
	if err != nil {
		return err
	}
	s.logger.SetLevel(level)
	return nil
}

func (s *Server) configureStore() error {
	st := store.New(s.config.Store)
	if err := st.Open(); err != nil {
		return err
	}
	s.store = st
	return nil
}

func (s *Server) configureBot() error {
	bot, err := tgbotapi.NewBotAPI(s.config.BotToken)
	if err != nil {
		return err
	}
	bot.Debug = true
	s.bot = bot
	log.Printf("Authorized on account %s", bot.Self.UserName)
	return nil
}

func (s *Server) configureBotUpdates() {
	s.updatesConf = tgbotapi.NewUpdate(0)
	s.updatesConf.Timeout = 60
}

func isGroupNumber(s *Server, userGroup string) bool {
	matched, err := regexp.Match(`^\d\d\d\d\d$`, []byte(userGroup))
	if err != nil {
		s.logger.Error(err)
	}
	return matched
}
