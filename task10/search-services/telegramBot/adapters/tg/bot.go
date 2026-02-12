package tg

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"yadro.com/course/telegramBot/adapters/rest"
)

// Признаю, сильно сделано
const (
	btnSearch = "🔎 Search"
	btnHelp   = "ℹ️ Help"
	btnStatus = "⏳ Status"
	btnStats  = "📊 Stats"
	btnUpdate = "🔄 Update"
	btnDrop   = "🗑 Drop"
)

type Bot struct {
	api     *rest.Client
	bot     *tgbotapi.BotAPI
	adminID int64
}

func NewBot(token string, apiClient *rest.Client, adminID int64) (*Bot, error) {
	botAPI, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("cannot create bot: %w", err)
	}
	botAPI.Debug = true

	return &Bot{
		api:     apiClient,
		bot:     botAPI,
		adminID: adminID,
	}, nil
}

func (b *Bot) Run(ctx context.Context) error {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.bot.GetUpdatesChan(u)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case update, ok := <-updates:
			if !ok {
				return nil
			}
			if update.Message == nil {
				continue
			}
			b.handleMessage(update.Message)
		}
	}
}

func (b *Bot) sendMenu(chatID int64, isAdmin bool) {
	rows := [][]tgbotapi.KeyboardButton{
		{
			tgbotapi.NewKeyboardButton(btnSearch),
			tgbotapi.NewKeyboardButton(btnHelp),
		},
		{
			tgbotapi.NewKeyboardButton(btnStatus),
			tgbotapi.NewKeyboardButton(btnStats),
		},
	}

	if isAdmin {
		rows = append(rows,
			[]tgbotapi.KeyboardButton{
				tgbotapi.NewKeyboardButton(btnUpdate),
				tgbotapi.NewKeyboardButton(btnDrop),
			},
		)
	}

	kb := tgbotapi.NewReplyKeyboard(rows...)
	kb.ResizeKeyboard = true
	kb.OneTimeKeyboard = false

	m := tgbotapi.NewMessage(chatID, "Меню команд:")
	m.ReplyMarkup = kb

	if _, err := b.bot.Send(m); err != nil {
		log.Println("send menu error:", err)
	}
}

func (b *Bot) handleMessage(msg *tgbotapi.Message) {
	chatID := msg.Chat.ID

	if b.handleButton(chatID, msg) {
		return
	}

	if msg.IsCommand() {
		cmd := msg.Command()
		args := msg.CommandArguments()

		switch cmd {
		case "start":
			text := "Привет! Я бот к XKCD-поисковому сервису.\n" +
				"Используй кнопки меню или /help."
			b.send(chatID, text)
			b.sendMenu(chatID, b.isAdmin(msg))

		case "help":
			isAdmin := b.isAdmin(msg)
			b.send(chatID, b.helpText(isAdmin))
			b.sendMenu(chatID, isAdmin)

		case "search":
			b.handleSearch(msg, args)

		case "update":
			if !b.isAdmin(msg) {
				b.send(chatID, "Эта команда доступна только администратору.")
				return
			}
			b.handleUpdate(msg)

		case "status":
			b.handleStatus(msg)

		case "stats":
			b.handleStats(msg)

		case "drop":
			if !b.isAdmin(msg) {
				b.send(chatID, "Эта команда доступна только администратору.")
				return
			}
			b.handleDrop(msg)

		default:
			b.send(chatID, "Неизвестная команда. Используй /help.")
		}
		return
	}

	// любой текст без слеша считаем поисковым запросом
	if strings.TrimSpace(msg.Text) != "" {
		b.handleSearch(msg, msg.Text)
	}
}

func (b *Bot) handleButton(chatID int64, msg *tgbotapi.Message) bool {
	text := strings.TrimSpace(msg.Text)
	if text == "" {
		return false
	}

	switch text {
	case btnHelp:
		isAdmin := b.isAdmin(msg)
		b.send(chatID, b.helpText(isAdmin))
		b.sendMenu(chatID, isAdmin)
		return true

	case btnStats:
		b.handleStats(msg)
		return true

	case btnStatus:
		b.handleStatus(msg)
		return true

	case btnUpdate:
		b.handleUpdate(msg)
		return true

	case btnDrop:
		b.handleDrop(msg)
		return true

	case btnSearch:
		b.send(chatID, "Введите фразу для поиска (можно просто текстом, без /search).\nНапример: linux")
		return true

	default:
		return false
	}
}

func helpTextUser() string {
	return `Доступные команды:
/start - Приветственное сообщение
/help - Вывести список доступных команд
/search <фраза> - поиск комиксов по фразе

Также можно просто отправить текст в чат и выполнится поиск`
}

func helpTextAdmin() string {
	return helpTextUser() + `
Всякие админские команды:
/update - запустить обновление базы комиксов
/status - статус обновления
/stats - статистика по базе
/drop - очистить базу`
}

func (b *Bot) helpText(isAdmin bool) string {
	if isAdmin {
		return helpTextAdmin()
	}
	return helpTextUser()
}

func (b *Bot) isAdmin(msg *tgbotapi.Message) bool {
	if b.adminID == 0 {
		// Если не задан, считаем, что ограничений нет
		return true
	}
	if msg.From == nil {
		return false
	}
	return msg.From.ID == b.adminID
}

func (b *Bot) handleSearch(msg *tgbotapi.Message, args string) {
	chatID := msg.Chat.ID
	phrase := strings.TrimSpace(args)

	if phrase == "" {
		b.send(chatID, "Укажите поисковую фразу: /search linux")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// По умолчанию, тем более для пользователя, делаем поиск по индексу
	res, err := b.api.IndexSearch(ctx, phrase, 5)
	if err != nil {
		b.send(chatID, "Ошибка поиска: "+err.Error())
		return
	}

	if len(res.Comics) == 0 {
		b.send(chatID, "Ничего не найдено по запросу: "+phrase)
		return
	}

	// Сначала короткое сообщение
	b.send(chatID, fmt.Sprintf("Найдено %d (показываю %d):", res.Total, len(res.Comics)))

	// Потом картинки
	for i, c := range res.Comics {
		caption := fmt.Sprintf("#%d (%d/%d)\n%s", c.ID, i+1, len(res.Comics), phrase)

		photo := tgbotapi.NewPhoto(chatID, tgbotapi.FileURL(c.URL))
		photo.Caption = caption

		if _, err := b.bot.Send(photo); err != nil {
			b.send(chatID, fmt.Sprintf("Не удалось отправить #%d: %v\n%s", c.ID, err, c.URL))
		}
	}
}

func (b *Bot) handleUpdate(msg *tgbotapi.Message) {
	chatID := msg.Chat.ID

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	resp, err := b.api.Update(ctx)
	if err != nil {
		b.send(chatID, "Ошибка запуска обновления: "+err.Error())
		return
	}
	b.send(chatID, resp)
}

func (b *Bot) handleStatus(msg *tgbotapi.Message) {
	chatID := msg.Chat.ID

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	st, err := b.api.Status(ctx)
	if err != nil {
		b.send(chatID, "Ошибка получения статуса: "+err.Error())
		return
	}

	b.send(chatID, "Статус обновления: "+st.Status)
}

func (b *Bot) handleStats(msg *tgbotapi.Message) {
	chatID := msg.Chat.ID

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	st, err := b.api.Stats(ctx)
	if err != nil {
		b.send(chatID, "Ошибка получения статистики: "+err.Error())
		return
	}

	text := fmt.Sprintf(
		"Статистика:\nВсего слов: %d\nУникальных слов: %d\nКомиксов в БД: %d\nКомиксов всего: %d",
		st.WordsTotal, st.WordsUnique, st.ComicsFetched, st.ComicsTotal,
	)

	b.send(chatID, text)
}

func (b *Bot) handleDrop(msg *tgbotapi.Message) {
	chatID := msg.Chat.ID

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := b.api.Drop(ctx); err != nil {
		b.send(chatID, "Ошибка очистки БД: "+err.Error())
		return
	}

	b.send(chatID, "База очищена.")
}

func (b *Bot) send(chatID int64, text string) {
	m := tgbotapi.NewMessage(chatID, text)
	m.ParseMode = "Markdown"
	if _, err := b.bot.Send(m); err != nil {
		log.Println("send error:", err)
	}
}
