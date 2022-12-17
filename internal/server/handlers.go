package server

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"net"
	"net/http"
	constants "reminder/internal"
	"time"
)

// handleStartCommand ...
func (ms *Microservice) handleStartCommand(update tgbotapi.Update) {
	if err := ms.store.AddUser(update.Message.Chat.ID); err != nil {
		ms.logger.Error(err)
	}

	msgText := constants.StartCommandMessage
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, msgText)
	if _, err := ms.bot.Send(msg); err != nil {
		ms.logger.Error(err)
	}
}

// handleHelpCommand ...
func (ms *Microservice) handleHelpCommand(update tgbotapi.Update) {
	msgText := constants.HelpCommandMessage
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, msgText)
	if _, err := ms.bot.Send(msg); err != nil {
		ms.logger.Error(err)
	}
}

// handleChangeGroupCommand
func (ms *Microservice) handleChangeGroupCommand(update tgbotapi.Update) {
	if len(update.Message.CommandArguments()) == 0 {
		msgText := constants.ChangeGroupCommandMessage
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, msgText)
		if _, err := ms.bot.Send(msg); err != nil {
			ms.logger.Error(err)
		}
	}
	ms.logger.Println(update.Message.CommandArguments() + "!!!!!!!!!!")
	//@TODO change group here
}

// handleDefaultCommand
func (ms *Microservice) handleDefaultCommand(update tgbotapi.Update) {
	//@TODO implement a default command handler for unsupported commands
}

// handleDefaultMessage Processes a message that did not pass the condition for processing by any other handler
func (ms *Microservice) handleDefaultMessage(update tgbotapi.Update) {
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
	hasGroup, err := ms.store.UserInGroup(update.Message.From.ID)
	if err != nil {
		ms.logger.Error(err)
		return
	}

	if hasGroup {
		msg.Text = constants.HasGroupMessage
		ms.bot.Send(msg)
		return
	}

	if isGroupNumber(ms, update.Message.Text) {
		if _, err := net.DialTimeout("tcp", "mysyte:myport", time.Second); err != nil {
			msg.Text = constants.ScheduleNotAvailableMessage
			ms.bot.Send(msg)
			return
		}

		ms.getScheduleAndAttachToUser(update)
	} else {
		msg.Text = constants.IncorrectGroupNumberMessage
		ms.bot.Send(msg)
		return
	}
}

func (ms *Microservice) getScheduleAndAttachToUser(update tgbotapi.Update) {
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")

	resp, err := http.Get("https://table.nsu.ru/group/" + update.Message.Text)
	if err != nil || resp.StatusCode == 404 {
		msg.Text = constants.NoScheduleFoundMessage
		ms.bot.Send(msg)
		return
	}
	exists, err := ms.store.ScheduleExists(update.Message.Text)
	if err != nil {
		msg.Text = constants.InternalErrorMessage
		ms.bot.Send(msg)
		ms.logger.Error(err)
		return
	}
	if !exists {
		schedule, err := Parse(resp, "div.subject", "div.room a")
		if err != nil {
			msg.Text = constants.InternalErrorMessage
			ms.bot.Send(msg)
			ms.logger.Error(err)
			return
		}
		if err := ms.store.AddSchedule(update.Message.Text, schedule); err != nil {
			msg.Text = constants.InternalErrorMessage
			ms.bot.Send(msg)
			ms.logger.Error(err)
			return
		}
	}
	if err := ms.store.AddUserToSchedule(update.Message.From.ID, update.Message.Text); err != nil {
		msg.Text = constants.InternalErrorMessage
		ms.bot.Send(msg)
		ms.logger.Error(err)
		return
	}
	// @TODO recheck this function
	if err := ms.store.UpdateUserHasGroup(update.Message.From.ID, 1); err != nil {
		msg.Text = constants.InternalErrorMessage
		ms.bot.Send(msg)
		ms.logger.Error(err)
		return
	}
	msg.Text = constants.UserAddedToScheduleMessage + update.Message.Text
	ms.bot.Send(msg)
	return
}
