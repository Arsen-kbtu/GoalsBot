package repository

import (
	"awesomeProject/internal/models"
	"database/sql"
	"time"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// User methods
func (r *Repository) GetOrCreateUser(tgID int64, username string) (*models.User, error) {
	var user models.User
	err := r.db.QueryRow(`
		SELECT id, tg_id, username, balance, created_at 
		FROM users WHERE tg_id = $1
	`, tgID).Scan(&user.ID, &user.TgID, &user.Username, &user.Balance, &user.CreatedAt)

	if err == sql.ErrNoRows {
		err = r.db.QueryRow(`
			INSERT INTO users (tg_id, username, balance) 
			VALUES ($1, $2, 100) 
			RETURNING id, tg_id, username, balance, created_at
		`, tgID, username).Scan(&user.ID, &user.TgID, &user.Username, &user.Balance, &user.CreatedAt)
	}

	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *Repository) UpdateUserBalance(userID int, amount int) error {
	_, err := r.db.Exec(`UPDATE users SET balance = balance + $1 WHERE id = $2`, amount, userID)
	return err
}

func (r *Repository) GetUserBalance(userID int) (int, error) {
	var balance int
	err := r.db.QueryRow(`SELECT balance FROM users WHERE id = $1`, userID).Scan(&balance)
	return balance, err
}

// Goal methods
func (r *Repository) CreateGoal(userID int, chatID int64, title, description string, deadline time.Time, bet int) (*models.Goal, error) {
	var goal models.Goal
	err := r.db.QueryRow(`
		INSERT INTO goals (user_id, chat_id, title, description, deadline, bet, status) 
		VALUES ($1, $2, $3, $4, $5, $6, 'active') 
		RETURNING id, user_id, title, description, deadline, bet, status, created_at
	`, userID, chatID, title, description, deadline, bet).Scan(
		&goal.ID, &goal.UserID, &goal.Title, &goal.Description,
		&goal.Deadline, &goal.Bet, &goal.Status, &goal.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &goal, nil
}

func (r *Repository) GetGoal(goalID int) (*models.Goal, error) {
	var goal models.Goal
	err := r.db.QueryRow(`
		SELECT id, user_id, title, description, deadline, bet, status, created_at 
		FROM goals WHERE id = $1
	`, goalID).Scan(&goal.ID, &goal.UserID, &goal.Title, &goal.Description,
		&goal.Deadline, &goal.Bet, &goal.Status, &goal.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &goal, nil
}

func (r *Repository) UpdateGoalStatus(goalID int, status string) error {
	_, err := r.db.Exec(`UPDATE goals SET status = $1 WHERE id = $2`, status, goalID)
	return err
}

func (r *Repository) UpdateGoalProof(goalID int, proof string) error {
	_, err := r.db.Exec(`UPDATE goals SET proof_message = $1, status = 'done_pending' WHERE id = $2`, proof, goalID)
	return err
}

func (r *Repository) GetActiveGoalsByChat(chatID int64) ([]models.Goal, error) {
	rows, err := r.db.Query(`
		SELECT id, user_id, title, description, deadline, bet, status, created_at 
		FROM goals WHERE chat_id = $1 AND status IN ('active', 'done_pending')
		ORDER BY created_at DESC
	`, chatID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var goals []models.Goal
	for rows.Next() {
		var goal models.Goal
		if err := rows.Scan(&goal.ID, &goal.UserID, &goal.Title, &goal.Description,
			&goal.Deadline, &goal.Bet, &goal.Status, &goal.CreatedAt); err != nil {
			return nil, err
		}
		goals = append(goals, goal)
	}
	return goals, nil
}

func (r *Repository) GetUserActiveGoals(userID int) ([]models.Goal, error) {
	rows, err := r.db.Query(`
		SELECT id, user_id, title, description, deadline, bet, status, created_at 
		FROM goals WHERE user_id = $1 AND status IN ('active', 'done_pending')
		ORDER BY deadline ASC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var goals []models.Goal
	for rows.Next() {
		var goal models.Goal
		if err := rows.Scan(&goal.ID, &goal.UserID, &goal.Title, &goal.Description,
			&goal.Deadline, &goal.Bet, &goal.Status, &goal.CreatedAt); err != nil {
			return nil, err
		}
		goals = append(goals, goal)
	}
	return goals, nil
}

// Vote methods
func (r *Repository) CreateVote(goalID, voterID int, vote bool) error {
	_, err := r.db.Exec(`
		INSERT INTO votes (goal_id, voter_id, vote) 
		VALUES ($1, $2, $3)
		ON CONFLICT (goal_id, voter_id) DO UPDATE SET vote = $3
	`, goalID, voterID, vote)
	return err
}

func (r *Repository) GetVotesByGoal(goalID int) ([]models.Vote, error) {
	rows, err := r.db.Query(`
		SELECT id, goal_id, voter_id, vote, created_at 
		FROM votes WHERE goal_id = $1
	`, goalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var votes []models.Vote
	for rows.Next() {
		var vote models.Vote
		if err := rows.Scan(&vote.ID, &vote.GoalID, &vote.VoterID, &vote.Vote, &vote.CreatedAt); err != nil {
			return nil, err
		}
		votes = append(votes, vote)
	}
	return votes, nil
}

func (r *Repository) CountVotes(goalID int) (yesCount int, noCount int, err error) {
	err = r.db.QueryRow(`
		SELECT 
			COUNT(CASE WHEN vote = true THEN 1 END) as yes_count,
			COUNT(CASE WHEN vote = false THEN 1 END) as no_count
		FROM votes WHERE goal_id = $1
	`, goalID).Scan(&yesCount, &noCount)
	return
}

// ChatMember methods
func (r *Repository) AddChatMember(chatID int64, userID int) error {
	_, err := r.db.Exec(`
		INSERT INTO chat_members (chat_id, user_id) 
		VALUES ($1, $2) 
		ON CONFLICT (chat_id, user_id) DO NOTHING
	`, chatID, userID)
	return err
}

func (r *Repository) GetChatMembers(chatID int64) ([]models.User, error) {
	rows, err := r.db.Query(`
		SELECT u.id, u.tg_id, u.username, u.balance, u.created_at 
		FROM users u
		INNER JOIN chat_members cm ON u.id = cm.user_id
		WHERE cm.chat_id = $1
	`, chatID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		if err := rows.Scan(&user.ID, &user.TgID, &user.Username, &user.Balance, &user.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, nil
}

// Transaction methods
func (r *Repository) CreateTransaction(fromUserID, toUserID, amount int, reason string, goalID *int) error {
	_, err := r.db.Exec(`
		INSERT INTO transactions (from_user_id, to_user_id, amount, reason, goal_id) 
		VALUES ($1, $2, $3, $4, $5)
	`, fromUserID, toUserID, amount, reason, goalID)
	return err
}
