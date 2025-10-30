package models

import "time"

// User represents a user in the system.
type User struct {
	ID        int       // Internal ID
	TgID      int64     // Telegram ID of the user
	Username  string    // @nickname or name
	Balance   int       // Balance of "stars"
	CreatedAt time.Time // When the user was added to the system
}

// Goal represents a goal created by a user.
type Goal struct {
	ID               int       // Goal ID
	UserID           int       // Author of the goal (foreign key to users.id)
	Title            string    // Title of the goal
	Description      string    // Description of the goal
	Deadline         time.Time // Deadline for the goal
	Bet              int       // Number of "stars" as penalty
	Status           string    // Status: active / done_pending / success / failed
	CreatedAt        time.Time // When the goal was created
	VotingStartedAt  *time.Time
	ChatMembersCount int
}

// Vote represents a vote on a goal.
type Vote struct {
	ID        int       // Vote ID
	GoalID    int       // Reference to the goal (foreign key to goals.id)
	VoterID   int       // Who voted (foreign key to users.id)
	Vote      bool      // True for confirmation, false otherwise
	CreatedAt time.Time // When the vote was cast
}

// Transaction represents a transaction of "stars".
type Transaction struct {
	ID        int       // Transaction ID
	FromUser  int       // From whom the stars were deducted (foreign key to users.id)
	ToUser    int       // To whom the stars were added (foreign key to users.id)
	Amount    int       // Number of stars transferred
	Reason    string    // Reason for the transaction (penalty, reward, transfer)
	CreatedAt time.Time // Date of the transaction
}
