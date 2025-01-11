-- Drop tables if they exist to avoid conflicts
DROP TABLE IF EXISTS BorrowedBooks;
DROP TABLE IF EXISTS Books;
DROP TABLE IF EXISTS Users;

-- Create Users table
CREATE TABLE Users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(100) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL,
    role     VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    jwt_token TEXT
);

-- Create Books table
CREATE TABLE Books (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(255) NOT NULL,
    author VARCHAR(255) NOT NULL,
    published_date TIMESTAMP NOT NULL,
    status VARCHAR(50) DEFAULT 'Available' NOT NULL,
    user_id UUID REFERENCES Users(id) ON DELETE SET NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create the BorrowedBooks table
CREATE TABLE BorrowedBooks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    book_id UUID NOT NULL REFERENCES Books(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES Users(id) ON DELETE CASCADE,
    borrowed_date TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    return_date TIMESTAMP
);

-- Insert three dummy users into the Users table
INSERT INTO Users (username, password, role)
VALUES
('user1', 'hashed_password_1', 'admin'),
('user2', 'hashed_password_2', 'user'),
('user3', 'hashed_password_3', 'user');

-- Insert sample books into the Books table
INSERT INTO Books (title, author, published_date, status, user_id)
VALUES
('The Great Gatsby', 'F. Scott Fitzgerald', '1925-04-10 00:00:00', 'Available', NULL),
('1984', 'George Orwell', '1949-06-08 00:00:00', 'Borrowed', (SELECT id FROM Users WHERE username = 'user1')),
('To Kill a Mockingbird', 'Harper Lee', '1960-07-11 00:00:00', 'Available', NULL),
('Pride and Prejudice', 'Jane Austen', '1813-01-28 00:00:00', 'Borrowed', (SELECT id FROM Users WHERE username = 'user2')),
('Moby-Dick', 'Herman Melville', '1851-10-18 00:00:00', 'Available', NULL);

-- Insert borrowed books into the BorrowedBooks table
INSERT INTO BorrowedBooks (book_id, user_id, borrowed_date, return_date)
VALUES
((SELECT id FROM Books WHERE title = '1984'),
 (SELECT id FROM Users WHERE username = 'user1'),
 '2025-01-01 10:00:00',
 NULL), -- User1 borrowed '1984' and has not yet returned it

((SELECT id FROM Books WHERE title = 'Pride and Prejudice'),
 (SELECT id FROM Users WHERE username = 'user2'),
 '2025-01-02 14:30:00',
 '2025-01-05 15:00:00'); -- User2 borrowed 'Pride and Prejudice' and returned it
