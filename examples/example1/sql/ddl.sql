CREATE TABLE user (
    id INT AUTO_INCREMENT PRIMARY KEY,
    fill_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    nick VARCHAR(64) NOT NULL DEFAULT '',
    gender ENUM('male', 'female', '') DEFAULT NULL,
    tag SET('a', 'b', 'c', '', 'd', 'x') DEFAULT NULL
);

CREATE TABLE blog (
    id INT AUTO_INCREMENT PRIMARY KEY,
    fill_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    user_id INT NOT NULL,
    title VARCHAR(256) NOT NULL,
    content TEXT,
    FOREIGN KEY (user_id) REFERENCES user (id)
);

