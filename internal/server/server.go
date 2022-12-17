package server

import (
	"github.com/PuerkitoBio/goquery"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
	"log"
	"net/http"
	"regexp"
	"reminder/pkg/store"
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

// Microservice ...
type Microservice struct {
	config      *Config
	logger      *logrus.Logger
	store       *store.Store
	bot         *tgbotapi.BotAPI
	updatesConf tgbotapi.UpdateConfig
}

// New ...
func New(config *Config) *Microservice {
	return &Microservice{
		config: config,
		logger: logrus.New(),
	}
}

// Start ...
func (ms *Microservice) Start() error {
	if err := ms.configureLogger(); err != nil {
		return err
	}

	if err := ms.configureStore(); err != nil {
		return err
	}

	if err := ms.configureBot(); err != nil {
		return err
	}

	ms.configureBotUpdates()
	ms.logger.Info("Telegram bot started!")

	ms.handleBotUpdates()

	return nil
}

// handleBotUpdates ...
func (ms *Microservice) handleBotUpdates() {
	updates := ms.bot.GetUpdatesChan(ms.updatesConf)
	for update := range updates {
		if update.Message != nil {
			ms.logger.Info("Incomig message: " + update.Message.Text)

			switch update.Message.Command() {
			case "start":
				ms.handleStartCommand(update)
				continue

			case "help":
				ms.handleHelpCommand(update)
				continue

			case "change":
				ms.handleChangeGroupCommand(update)
				continue
			default:
				ms.handleDefaultCommand(update)
			}

			switch update.Message.Text {
			case "open":
				// @TODO Handle "open" command here

			default:
				ms.handleDefaultMessage(update)

			}
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

func (ms *Microservice) configureLogger() error {
	level, err := logrus.ParseLevel(ms.config.LogLevel)
	if err != nil {
		return err
	}
	ms.logger.SetLevel(level)
	return nil
}

func (ms *Microservice) configureStore() error {
	st := store.New(ms.config.Store)
	if err := st.Open(); err != nil {
		return err
	}
	ms.store = st
	return nil
}

func (ms *Microservice) configureBot() error {
	bot, err := tgbotapi.NewBotAPI(ms.config.BotToken)
	if err != nil {
		return err
	}
	bot.Debug = false
	ms.bot = bot
	log.Printf("Authorized on account %ms", bot.Self.UserName)
	return nil
}

func (ms *Microservice) configureBotUpdates() {
	ms.updatesConf = tgbotapi.NewUpdate(0)
	ms.updatesConf.Timeout = 60
}

func isGroupNumber(s *Microservice, userGroup string) bool {
	matched, err := regexp.Match(`^\d\d\d\d\d$`, []byte(userGroup))
	if err != nil {
		s.logger.Error(err)
	}
	return matched
}
