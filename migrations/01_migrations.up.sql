CREATE TABLE users(
    id SERIAL PRIMARY KEY,
    tg_id BIGINT UNIQUE NOT NULL,
    username VARCHAR(255),
    balance INT DEFAULT 100,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE goals(
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL,
    chat_id BIGINT NOT NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    deadline TIMESTAMP NOT NULL,
    bet INT NOT NULL,
    status VARCHAR(50) DEFAULT 'active',
    proof_message TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE votes(
    id SERIAL PRIMARY KEY,
    goal_id INT NOT NULL,
    voter_id INT NOT NULL,
    vote BOOLEAN NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (goal_id) REFERENCES goals(id) ON DELETE CASCADE,
    FOREIGN KEY (voter_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE(goal_id, voter_id)
);

CREATE TABLE transactions(
    id SERIAL PRIMARY KEY,
    from_user_id INT,
    to_user_id INT,
    amount INT NOT NULL,
    reason VARCHAR(100) NOT NULL,
    goal_id INT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (from_user_id) REFERENCES users(id) ON DELETE SET NULL,
    FOREIGN KEY (to_user_id) REFERENCES users(id) ON DELETE SET NULL,
    FOREIGN KEY (goal_id) REFERENCES goals(id) ON DELETE SET NULL
);

CREATE TABLE chat_members(
    id SERIAL PRIMARY KEY,
    chat_id BIGINT NOT NULL,
    user_id INT NOT NULL,
    joined_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE(chat_id, user_id)
);