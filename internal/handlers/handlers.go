package handlers

import (
	"awesomeProject/internal/models"
	"awesomeProject/internal/service"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
	"strconv"
	"strings"
	"time"
)

type BotHandler struct {
	bot        *tgbotapi.BotAPI
	service    *service.Service
	userStates map[int64]*UserState
}

type UserState struct {
	Step        string
	GoalData    *models.Goal
	Title       string
	Description string
	Deadline    time.Time
	Bet         int
}

func NewBotHandler(bot *tgbotapi.BotAPI, service *service.Service) *BotHandler {
	return &BotHandler{
		bot:        bot,
		service:    service,
		userStates: make(map[int64]*UserState),
	}
}

func (h *BotHandler) HandleUpdate(update tgbotapi.Update) {
	// Handle messages
	if update.Message != nil {
		h.handleMessage(update.Message)
	}

	// Handle callback queries (buttons)
	if update.CallbackQuery != nil {
		h.handleCallbackQuery(update.CallbackQuery)
	}
}

func (h *BotHandler) handleMessage(message *tgbotapi.Message) {
	// Register user
	username := message.From.UserName
	if username == "" {
		username = message.From.FirstName
	}

	user, err := h.service.RegisterUser(message.From.ID, username, message.Chat.ID)
	if err != nil {
		log.Printf("Error registering user: %v", err)
	}

	// Handle commands
	if message.IsCommand() {
		switch message.Command() {
		case "start":
			h.handleStart(message)
		case "help":
			h.handleHelp(message)
		case "newgoal":
			h.handleNewGoal(message, user)
		case "mygoals":
			h.handleMyGoals(message, user)
		case "goals":
			h.handleChatGoals(message)
		case "stats":
			h.handleStats(message, user)
		case "cancel":
			h.handleCancel(message)
		}
		return
	}

	// Handle state-based input
	if state, exists := h.userStates[message.From.ID]; exists {
		h.handleStateInput(message, state, user)
	}
}

func (h *BotHandler) handleStart(message *tgbotapi.Message) {
	text := `👋 Привет! Я бот для постановки целей с ответственностью.

🎯 Как это работает:
1. Создай цель с помощью /newgoal
2. Укажи название, описание, срок и ставку в звездах
3. После достижения цели отправь доказательство
4. Участники беседы проголосуют за выполнение
5. Если цель выполнена - ты сохраняешь звезды
6. Если нет - звезды распределятся между участниками

📋 Команды:
/newgoal - Создать новую цель
/mygoals - Мои активные цели
/goals - Все цели в беседе
/stats - Моя статистика
/help - Помощь`

	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	h.bot.Send(msg)
}

func (h *BotHandler) handleHelp(message *tgbotapi.Message) {
	text := `📖 Помощь

📋 Команды:
/newgoal - Создать новую цель
/mygoals - Посмотреть свои активные цели
/goals - Посмотреть все цели в беседе
/stats - Моя статистика
/cancel - Отменить текущее действие

💡 Советы:
• Ставьте реалистичные цели
• Сохраняйте доказательства выполнения
• Голосуйте честно за цели других участников
• Начальный баланс: 100 звезд`

	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	h.bot.Send(msg)
}

func (h *BotHandler) handleNewGoal(message *tgbotapi.Message, _ *models.User) {
	h.userStates[message.From.ID] = &UserState{
		Step: "awaiting_title",
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, "📝 Введите название цели:")
	_, _ = h.bot.Send(msg)
}

func (h *BotHandler) handleStateInput(message *tgbotapi.Message, state *UserState, user *models.User) {
	switch state.Step {
	case "awaiting_title":
		state.Title = message.Text
		state.Step = "awaiting_description"
		msg := tgbotapi.NewMessage(message.Chat.ID, "📄 Введите описание цели:")
		_, _ = h.bot.Send(msg)

	case "awaiting_description":
		state.Description = message.Text
		state.Step = "awaiting_deadline"
		msg := tgbotapi.NewMessage(message.Chat.ID, "📅 Введите срок выполнения (формат: 2024-12-31 или количество дней, например: 7):")
		_, _ = h.bot.Send(msg)

	case "awaiting_deadline":
		deadline, err := h.parseDeadline(message.Text)
		if err != nil {
			msg := tgbotapi.NewMessage(message.Chat.ID, "❌ Неверный формат даты. Используйте формат YYYY-MM-DD или количество дней (например: 7)")
			_, _ = h.bot.Send(msg)
			return
		}
		state.Deadline = deadline
		state.Step = "awaiting_bet"

		// Get fresh user data to show current balance
		freshUser, err := h.service.GetOrCreateUser(message.From.ID, message.From.UserName)
		if err != nil {
			log.Printf("Error getting user: %v", err)
			msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("⭐ Введите ставку в звездах:"))
			_, _ = h.bot.Send(msg)
		} else {
			msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("⭐ Введите ставку в звездах (ваш баланс: %d):", freshUser.Balance))
			_, _ = h.bot.Send(msg)
		}

	case "awaiting_bet":
		bet, err := strconv.Atoi(message.Text)
		if err != nil || bet <= 0 {
			msg := tgbotapi.NewMessage(message.Chat.ID, "❌ Неверное значение. Введите положительное число:")
			_, _ = h.bot.Send(msg)
			return
		}

		// Get fresh user data to check current balance
		freshUser, err := h.service.GetOrCreateUser(message.From.ID, message.From.UserName)
		if err != nil {
			msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("❌ Ошибка получения данных пользователя: %v", err))
			_, _ = h.bot.Send(msg)
			delete(h.userStates, message.From.ID)
			return
		}

		if bet > freshUser.Balance {
			msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("❌ Недостаточно звезд. У вас: %d", freshUser.Balance))
			_, _ = h.bot.Send(msg)
			return
		}

		state.Bet = bet

		// Create goal - use freshUser.ID
		goal, err := h.service.CreateGoal(freshUser.ID, message.Chat.ID, state.Title, state.Description, state.Deadline, state.Bet)
		if err != nil {
			msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("❌ Ошибка создания цели: %v", err))
			_, _ = h.bot.Send(msg)
			delete(h.userStates, message.From.ID)
			return
		}

		text := fmt.Sprintf(`✅ Цель создана!

🎯 %s
📄 %s
📅 Срок: %s
⭐ Ставка: %d звезд

Удачи! После выполнения используйте команду /mygoals чтобы отправить доказательство.`,
			goal.Title,
			goal.Description,
			goal.Deadline.Format("02.01.2006"),
			goal.Bet,
		)

		msg := tgbotapi.NewMessage(message.Chat.ID, text)
		_, _ = h.bot.Send(msg)

		delete(h.userStates, message.From.ID)

	case "awaiting_proof":
		// Handle proof submission
		if state.GoalData != nil {
			err := h.service.SubmitProof(state.GoalData.ID, message.Text)
			if err != nil {
				msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("❌ Ошибка: %v", err))
				_, _ = h.bot.Send(msg)
				return
			}

			// Get fresh user data for username
			freshUser, err := h.service.GetOrCreateUser(message.From.ID, message.From.UserName)
			if err != nil {
				freshUser = user // fallback to cached user
			}

			// Send notification to chat with voting buttons
			text := fmt.Sprintf(`📢 @%s отправил доказательство выполнения цели:

🎯 %s
📄 %s
💬 Доказательство: %s

Голосуйте за выполнение:`,
				freshUser.Username,
				state.GoalData.Title,
				state.GoalData.Description,
				message.Text,
			)

			// Create inline keyboard with voting buttons
			keyboard := tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("✅ Выполнено", fmt.Sprintf("vote_yes_%d", state.GoalData.ID)),
					tgbotapi.NewInlineKeyboardButtonData("❌ Не выполнено", fmt.Sprintf("vote_no_%d", state.GoalData.ID)),
				),
			)

			msg := tgbotapi.NewMessage(message.Chat.ID, text)
			msg.ReplyMarkup = keyboard
			_, _ = h.bot.Send(msg)

			delete(h.userStates, message.From.ID)
		}
	}
}

func (h *BotHandler) handleMyGoals(message *tgbotapi.Message, user *models.User) {
	goals, err := h.service.GetUserActiveGoals(user.ID)
	if err != nil {
		msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("❌ Ошибка: %v", err))
		_, _ = h.bot.Send(msg)
		return
	}

	if len(goals) == 0 {
		msg := tgbotapi.NewMessage(message.Chat.ID, "У вас нет активных целей. Создайте новую с помощью /newgoal")
		_, _ = h.bot.Send(msg)
		return
	}

	text := "📋 Ваши активные цели:\n\n"

	var buttons [][]tgbotapi.InlineKeyboardButton
	for i, goal := range goals {
		statusEmoji := "🔄"
		if goal.Status == "done_pending" {
			statusEmoji = "⏳"
		}

		text += fmt.Sprintf("%d. %s %s\n   📅 %s | ⭐ %d\n\n",
			i+1,
			statusEmoji,
			goal.Title,
			goal.Deadline.Format("02.01.2006"),
			goal.Bet,
		)

		if goal.Status == "active" {
			buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(
					fmt.Sprintf("✅ Отправить доказательство #%d", i+1),
					fmt.Sprintf("proof_%d", goal.ID),
				),
			))
		}
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	if len(buttons) > 0 {
		msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(buttons...)
	}
	_, _ = h.bot.Send(msg)
}

func (h *BotHandler) handleChatGoals(message *tgbotapi.Message) {
	goals, err := h.service.GetActiveGoalsByChat(message.Chat.ID)
	if err != nil {
		msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("❌ Ошибка: %v", err))
		_, _ = h.bot.Send(msg)
		return
	}

	if len(goals) == 0 {
		msg := tgbotapi.NewMessage(message.Chat.ID, "В этой беседе нет активных целей.")
		_, _ = h.bot.Send(msg)
		return
	}

	text := "📋 Активные цели в беседе:\n\n"
	for i, goal := range goals {
		// Get user info
		user, _ := h.service.GetUserByID(int64(goal.UserID))

		statusEmoji := "🔄"
		statusText := "Активна"
		if goal.Status == "done_pending" {
			statusEmoji = "⏳"
			statusText = "На голосовании"
		}

		text += fmt.Sprintf("%d. %s %s\n   👤 @%s\n   📅 %s | ⭐ %d | %s\n\n",
			i+1,
			statusEmoji,
			goal.Title,
			user.Username,
			goal.Deadline.Format("02.01.2006"),
			goal.Bet,
			statusText,
		)
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	_, _ = h.bot.Send(msg)
}

func (h *BotHandler) handleStats(message *tgbotapi.Message, user *models.User) {
	stats, err := h.service.GetUserStats(user.ID)
	if err != nil {
		msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("❌ Ошибка: %v", err))
		h.bot.Send(msg)
		return
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, stats)
	h.bot.Send(msg)
}

func (h *BotHandler) handleCancel(message *tgbotapi.Message) {
	delete(h.userStates, message.From.ID)
	msg := tgbotapi.NewMessage(message.Chat.ID, "❌ Действие отменено.")
	h.bot.Send(msg)
}

func (h *BotHandler) handleCallbackQuery(query *tgbotapi.CallbackQuery) {
	parts := strings.Split(query.Data, "_")

	if len(parts) < 2 {
		return
	}

	action := parts[0]

	// Get user
	username := query.From.UserName
	if username == "" {
		username = query.From.FirstName
	}
	user, err := h.service.RegisterUser(query.From.ID, username, query.Message.Chat.ID)
	if err != nil {
		log.Printf("Error registering user: %v", err)
		return
	}

	switch action {
	case "proof":
		// User wants to submit proof
		goalID, _ := strconv.Atoi(parts[1])
		goal, err := h.service.GetGoal(goalID)
		if err != nil {
			h.answerCallback(query, "❌ Цель не найдена")
			return
		}

		h.userStates[query.From.ID] = &UserState{
			Step:     "awaiting_proof",
			GoalData: goal,
		}

		msg := tgbotapi.NewMessage(query.Message.Chat.ID, "📝 Отправьте доказательство выполнения цели (текст, фото, ссылка):")
		_, _ = h.bot.Send(msg)
		h.answerCallback(query, "")

	case "vote":
		// User is voting
		if len(parts) < 3 {
			return
		}

		voteType := parts[1] // "yes" or "no"
		goalID, _ := strconv.Atoi(parts[2])

		vote := voteType == "yes"
		err := h.service.VoteOnGoal(goalID, user.ID, vote)
		if err != nil {
			h.answerCallback(query, fmt.Sprintf("❌ %v", err))
			return
		}

		// Check if voting is complete and finalize
		result, err := h.service.FinalizeGoal(goalID, query.Message.Chat.ID)
		if err != nil {
			h.answerCallback(query, "✅ Голос учтен")
			return
		}

		// Send result to chat
		msg := tgbotapi.NewMessage(query.Message.Chat.ID, result)
		_, _ = h.bot.Send(msg)

		h.answerCallback(query, "✅ Голос учтен")
	}
}

func (h *BotHandler) answerCallback(query *tgbotapi.CallbackQuery, text string) {
	callback := tgbotapi.NewCallback(query.ID, text)
	h.bot.Request(callback)
}

func (h *BotHandler) parseDeadline(input string) (time.Time, error) {
	// Try parsing as date
	t, err := time.Parse("2006-01-02", input)
	if err == nil {
		return t, nil
	}

	// Try parsing as number of days
	days, err := strconv.Atoi(input)
	if err == nil && days > 0 {
		return time.Now().AddDate(0, 0, days), nil
	}

	return time.Time{}, fmt.Errorf("неверный формат")
}
