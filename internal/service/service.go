package service

import (
	"awesomeProject/internal/models"
	"awesomeProject/internal/repository"
	"fmt"
	"time"
)

type Service struct {
	repo *repository.Repository
}

func NewService(repo *repository.Repository) *Service {
	return &Service{repo: repo}
}

// RegisterUser creates or retrieves a user
func (s *Service) RegisterUser(tgID int64, username string, chatID int64) (*models.User, error) {
	user, err := s.repo.GetOrCreateUser(tgID, username)
	if err != nil {
		return nil, err
	}

	// Add user to chat members
	if err := s.repo.AddChatMember(chatID, user.ID); err != nil {
		return nil, err
	}

	return user, nil
}

// CreateGoal creates a new goal for a user
func (s *Service) CreateGoal(userID int, chatID int64, title, description string, deadline time.Time, bet int) (*models.Goal, error) {
	// Check if user has enough balance
	balance, err := s.repo.GetUserBalance(userID)
	if err != nil {
		return nil, err
	}

	if balance < bet {
		return nil, fmt.Errorf("недостаточно звезд на балансе. У вас: %d, требуется: %d", balance, bet)
	}

	// Create goal
	goal, err := s.repo.CreateGoal(userID, chatID, title, description, deadline, bet)
	if err != nil {
		return nil, err
	}

	return goal, nil
}

// SubmitProof submits proof of goal completion
func (s *Service) SubmitProof(goalID int, proof string) error {
	goal, err := s.repo.GetGoal(goalID)
	if err != nil {
		return err
	}

	if goal.Status != "active" {
		return fmt.Errorf("цель должна быть активной для отправки доказательства")
	}

	return s.repo.UpdateGoalProof(goalID, proof)
}

// VoteOnGoal allows a user to vote on a goal
func (s *Service) VoteOnGoal(goalID, voterID int, vote bool) error {
	goal, err := s.repo.GetGoal(goalID)
	if err != nil {
		return err
	}

	if goal.Status != "done_pending" {
		return fmt.Errorf("голосование доступно только для целей в статусе 'done_pending'")
	}

	// User can't vote for their own goal
	if goal.UserID == voterID {
		return fmt.Errorf("вы не можете голосовать за свою собственную цель")
	}

	return s.repo.CreateVote(goalID, voterID, vote)
}

// FinalizeGoal finalizes a goal based on votes
func (s *Service) FinalizeGoal(goalID int, chatID int64) (string, error) {
	goal, err := s.repo.GetGoal(goalID)
	if err != nil {
		return "", err
	}

	if goal.Status != "done_pending" {
		return "", fmt.Errorf("цель должна быть в статусе 'done_pending'")
	}

	// Get chat members count (excluding goal creator)
	members, err := s.repo.GetChatMembers(chatID)
	if err != nil {
		return "", err
	}

	// Count votes
	yesCount, noCount, err := s.repo.CountVotes(goalID)
	if err != nil {
		return "", err
	}

	totalVoters := len(members) - 1        // excluding goal creator
	requiredVotes := (totalVoters + 1) / 2 // majority

	var resultMessage string

	// Check if majority voted yes
	if yesCount >= requiredVotes {
		// Success - goal completed
		err = s.repo.UpdateGoalStatus(goalID, "success")
		if err != nil {
			return "", err
		}
		resultMessage = fmt.Sprintf("✅ Цель выполнена! Голосов ЗА: %d, ПРОТИВ: %d", yesCount, noCount)
	} else if noCount > totalVoters-requiredVotes {
		// Failed - not enough yes votes
		err = s.FailGoal(goalID, chatID)
		if err != nil {
			return "", err
		}
		resultMessage = fmt.Sprintf("❌ Цель провалена! Голосов ЗА: %d, ПРОТИВ: %d. Штраф распределен между участниками.", yesCount, noCount)
	} else {
		resultMessage = fmt.Sprintf("⏳ Ожидаем больше голосов. ЗА: %d, ПРОТИВ: %d (требуется: %d)", yesCount, noCount, requiredVotes)
	}

	return resultMessage, nil
}

// FailGoal handles goal failure and distributes penalty
func (s *Service) FailGoal(goalID int, chatID int64) error {
	goal, err := s.repo.GetGoal(goalID)
	if err != nil {
		return err
	}

	// Get all chat members except goal creator
	members, err := s.repo.GetChatMembers(chatID)
	if err != nil {
		return err
	}

	var recipients []models.User
	for _, member := range members {
		if member.ID != goal.UserID {
			recipients = append(recipients, member)
		}
	}

	if len(recipients) == 0 {
		return fmt.Errorf("нет участников для распределения штрафа")
	}

	// Deduct penalty from goal creator
	err = s.repo.UpdateUserBalance(goal.UserID, -goal.Bet)
	if err != nil {
		return err
	}

	// Distribute penalty among other members
	amountPerPerson := goal.Bet / len(recipients)
	remainder := goal.Bet % len(recipients)

	for i, recipient := range recipients {
		amount := amountPerPerson
		if i < remainder {
			amount++ // distribute remainder
		}

		err = s.repo.UpdateUserBalance(recipient.ID, amount)
		if err != nil {
			return err
		}

		// Record transaction
		err = s.repo.CreateTransaction(goal.UserID, recipient.ID, amount, "penalty_distribution", &goalID)
		if err != nil {
			return err
		}
	}

	// Update goal status
	err = s.repo.UpdateGoalStatus(goalID, "failed")
	if err != nil {
		return err
	}

	return nil
}

// CheckExpiredGoals checks for expired goals and fails them
func (s *Service) CheckExpiredGoals() error {
	// This would be called by a cron job or ticker
	// Implementation depends on how you want to handle scheduled tasks
	return nil
}

// GetUserStats returns user statistics
func (s *Service) GetUserStats(userID int) (string, error) {
	user, err := s.repo.GetOrCreateUser(int64(userID), "")
	if err != nil {
		return "", err
	}

	goals, err := s.repo.GetUserActiveGoals(userID)
	if err != nil {
		return "", err
	}

	stats := fmt.Sprintf("👤 Статистика\n")
	stats += fmt.Sprintf("⭐ Баланс: %d звезд\n", user.Balance)
	stats += fmt.Sprintf("📋 Активных целей: %d\n", len(goals))

	return stats, nil
}

// Public methods to access repository
func (s *Service) GetUserActiveGoals(userID int) ([]models.Goal, error) {
	return s.repo.GetUserActiveGoals(userID)
}

func (s *Service) GetActiveGoalsByChat(chatID int64) ([]models.Goal, error) {
	return s.repo.GetActiveGoalsByChat(chatID)
}

func (s *Service) GetGoal(goalID int) (*models.Goal, error) {
	return s.repo.GetGoal(goalID)
}

func (s *Service) GetOrCreateUser(tgID int64, username string) (*models.User, error) {
	return s.repo.GetOrCreateUser(tgID, username)
}
