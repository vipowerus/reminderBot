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
	s.logger.Info("Telegram bot started!")

	s.handleBotUpdates()

	return nil
}

// handleStartCommand ...
func (s *Server) handleStartCommand(update tgbotapi.Update) {
	if err := s.store.AddUser(update.Message.Chat.ID); err != nil {
		s.logger.Error(err)
	}

	msgText := "Hello! Nice to meet you \nPlease enter the group number"
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, msgText)
	if _, err := s.bot.Send(msg); err != nil {
		s.logger.Error(err)
	}
}

// handleHelpCommand ...
func (s *Server) handleHelpCommand(update tgbotapi.Update) {
	msgText := "Someday there will be help"
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, msgText)
	if _, err := s.bot.Send(msg); err != nil {
		s.logger.Error(err)
	}
}

// handleChangeGroupCommand
func (s *Server) handleChangeGroupCommand(update tgbotapi.Update) {
	if len(update.Message.CommandArguments()) == 0 {
		msgText := "Please add the group number to the command"
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, msgText)
		if _, err := s.bot.Send(msg); err != nil {
			s.logger.Error(err)
		}
	}
	s.logger.Println(update.Message.CommandArguments() + "!!!!!!!!!!")
	//@TODO change group here
}

// handleDefaultCommand
func (s *Server) handleDefaultCommand(update tgbotapi.Update) {
	//@TODO implement a default command handler for unsupported commands
}

func (s *Server) handleDefaultMessage(update tgbotapi.Update) {
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
	hasGroup, err := s.store.UserInGroup(update.Message.From.ID)
	if err != nil {
		s.logger.Error(err)
	}

	if hasGroup {
		msg.Text = "У вас уже есть группа. Вы можете сменить ее с помощью команды '/change *****'"
		s.bot.Send(msg)
		return
	}
	if isGroupNumber(s, update.Message.Text) {
		resp, err := http.Get("https://table.nsu.ru/group/" + update.Message.Text)
		if err != nil || resp.StatusCode == 404 {
			msg.Text = "Не могу найти ваше расписание"
			s.bot.Send(msg)
			return
		}
		exists, err := s.store.ScheduleExists(update.Message.Text)
		if err != nil {
			msg.Text = "Я сламался" // rework and add variable to Text!!!
			s.bot.Send(msg)
			s.logger.Error(err)
			return
		}
		if !exists {
			schedule, err := Parse(resp, "div.subject", "div.room a")
			if err != nil {
				msg.Text = "Я сламался" // rework
				s.bot.Send(msg)
				s.logger.Error(err)
				return
			}
			if err := s.store.AddSchedule(update.Message.Text, schedule); err != nil {
				msg.Text = "Я сламался" // rework
				s.bot.Send(msg)
				s.logger.Error(err)
				return
			}
		}
		if err := s.store.AddUserToSchedule(update.Message.From.ID, update.Message.Text); err != nil {
			msg.Text = "Я сламался" // rework
			s.bot.Send(msg)
			s.logger.Error(err)
			return
		}
		if err := s.store.UpdateUserHasGroup(true, update.Message.From.ID); err != nil { // CHECK IT!!!!!!!!!!
			msg.Text = "Я сламался" // rework
			s.bot.Send(msg)
			s.logger.Error(err)
			return
		}
		msg.Text = "Вы добавлены на рассылку по расписанию группы " + update.Message.Text
		s.bot.Send(msg)
		return

	} else {
		msg.Text = "Некорректный номер группы. Я умею работать с такими номерами: [0-9][0-9][0-9][0-9][0-9]"
		s.bot.Send(msg)
		return
	}
}

// handleBotUpdates ...
func (s *Server) handleBotUpdates() {
	updates := s.bot.GetUpdatesChan(s.updatesConf)
	for update := range updates {
		if update.Message != nil {
			s.logger.Info("Incomig message: " + update.Message.Text)

			switch update.Message.Command() {
			case "start":
				s.handleStartCommand(update)
				continue

			case "help":
				s.handleHelpCommand(update)
				continue

			case "change":
				s.handleChangeGroupCommand(update)
				continue
			default:
				s.handleDefaultCommand(update)
			}

			switch update.Message.Text {
			case "open":
				//@TODO Handle "open" command here

			default:
				s.handleDefaultMessage(update)

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
