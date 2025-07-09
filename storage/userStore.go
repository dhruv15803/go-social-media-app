package storage

type User struct {
	Id          int     `db:"id" json:"id"`
	Email       string  `db:"email" json:"email"`
	Username    string  `db:"username" json:"username"`
	ImageUrl    *string `db:"image_url" json:"image_url"`
	Password    string  `db:"password" json:"-"`
	Bio         *string `db:"bio" json:"bio"`
	Location    *string `db:"location" json:"location"`
	DateOfBirth string  `db:"date_of_birth" json:"date_of_birth"`
	IsPublic    bool    `db:"is_public" json:"is_public"`
	CreatedAt   string  `db:"created_at" json:"created_at"`
	UpdatedAt   *string `db:"updated_at" json:"updated_at"`
}

func (s *Storage) GetUsersByEmailOrUsername(email string, username string) ([]User, error) {

	var users []User

	query := `SELECT id,email,username,image_url,password,bio,location,date_of_birth,is_public,
	created_at,updated_at FROM users WHERE email=$1 OR username=$2`

	if err := s.db.Select(&users, query, email, username); err != nil {
		return []User{}, err
	}

	return users, nil
}

func (s *Storage) CreateUser(email string, username string, password string, dateOfBirth string) (*User, error) {

	var newUser User

	query := `INSERT INTO users(email,username,password,date_of_birth) VALUES($1,$2,$3,$4) RETURNING 
	id,email,username,image_url,password,bio,location,date_of_birth,is_public,created_at,updated_at`

	row := s.db.QueryRowx(query, email, username, password, dateOfBirth)

	if err := row.StructScan(&newUser); err != nil {
		return nil, err
	}

	return &newUser, nil
}

func (s *Storage) GetUserByEmail(email string) (*User, error) {

	var user User

	query := `SELECT id,email,username,image_url,password,
	bio,location,date_of_birth,is_public,created_at,updated_at 
	FROM users WHERE email=$1`

	if err := s.db.Get(&user, query, email); err != nil {
		return nil, err
	}

	return &user, nil
}

func (s *Storage) GetUserByUsername(username string) (*User, error) {

	var user User

	query := `SELECT id,email,username,image_url,password,bio,location,date_of_birth,
	is_public,created_at,updated_at FROM users WHERE username=$1`

	if err := s.db.Get(&user, query, username); err != nil {
		return nil, err
	}

	return &user, nil
}

func (s *Storage) GetUserById(id int) (*User, error) {
	var user User

	query := `SELECT id,email,username,image_url,password,bio,location,date_of_birth,
	is_public,created_at,updated_at FROM users WHERE id=$1`

	if err := s.db.Get(&user, query, id); err != nil {
		return nil, err
	}

	return &user, nil
}

func (s *Storage) GetFollowers(userId int, skip int, limit int) ([]User, error) {

	var followers []User

	query := `SELECT 
	id,email,username,image_url,password,bio,location,date_of_birth,is_public,created_at,updated_at 
	FROM users 
	WHERE id IN 
	(SELECT follower_id 
	FROM follows WHERE following_id=$1 
	ORDER BY followed_at DESC 
	OFFSET $2 LIMIT $3)`

	rows, err := s.db.Queryx(query, userId, skip, limit)
	if err != nil {
		return []User{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var follower User

		if err := rows.StructScan(&follower); err != nil {
			return []User{}, err
		}

		followers = append(followers, follower)
	}

	return followers, nil
}

func (s *Storage) GetFollowersCount(userId int) (int, error) {

	var totalFollowersCount int

	query := `SELECT COUNT(follower_id) FROM follows WHERE following_id=$1`

	row := s.db.QueryRow(query, userId)

	if err := row.Scan(&totalFollowersCount); err != nil {
		return -1, err
	}

	return totalFollowersCount, nil
}

func (s *Storage) GetFollowings(userId int, skip int, limit int) ([]User, error) {

	var followings []User

	query := `SELECT 
	id,email,username,image_url,password,bio,location,date_of_birth,is_public,created_at,updated_at 
	FROM users 
	WHERE id IN 
	(SELECT following_id 
	FROM follows WHERE follower_id=$1
	ORDER BY followed_at DESC 
	OFFSET $2 LIMIT $3)`

	rows, err := s.db.Queryx(query, userId, skip, limit)
	if err != nil {
		return []User{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var following User

		if err := rows.StructScan(&following); err != nil {
			return []User{}, err
		}

		followings = append(followings, following)
	}

	return followings, nil
}

func (s *Storage) GetFollowingsCount(userId int) (int, error) {

	var totalFollowingsCount int

	query := `SELECT COUNT(following_id) FROM follows WHERE follower_id=$1`

	row := s.db.QueryRow(query, userId)

	if err := row.Scan(&totalFollowingsCount); err != nil {
		return -1, err
	}

	return totalFollowingsCount, nil
}

func (s *Storage) UpdateUser(userId int, username string, imageUrl string, bio string, location string, isPublic bool) (*User, error) {
	var updatedUser User

	query := `UPDATE users SET username=$1,image_url=$2,bio=$3,location=$4,is_public=$5 WHERE id=$6
	RETURNING id,email,username,image_url,password,bio,location,date_of_birth,is_public,created_at,
	updated_at`

	row := s.db.QueryRowx(query, username, imageUrl, bio, location, isPublic, userId)

	if err := row.StructScan(&updatedUser); err != nil {
		return nil, err
	}

	return &updatedUser, nil
}

func (s *Storage) GetUsersBySearchText(searchText string, skip int, limit int) ([]User, error) {

	type UserWithFollowerCount struct {
		User
		FollowersCount int `db:"followers_count"`
	}

	var results []User

	query := `SELECT u.id,u.email,u.username,u.image_url,u.password,bio,u.location,u.date_of_birth,u.is_public,u.created_at,u.updated_at,COUNT(f.follower_id) AS followers_count
FROM users AS u LEFT JOIN follows AS f ON f.following_id=u.id
WHERE u.username ILIKE $1
GROUP BY u.id,f.following_id
ORDER BY followers_count DESC , u.created_at DESC 
LIMIT $2 OFFSET $3`

	var searchParam string

	if searchText == "" {
		searchParam = ""
	} else {
		searchParam = "%" + searchText + "%"
	}
	rows, err := s.db.Queryx(query, searchParam, limit, skip)
	if err != nil {
		return []User{}, err
	}

	defer rows.Close()

	for rows.Next() {
		var temp UserWithFollowerCount

		if err := rows.StructScan(&temp); err != nil {
			return []User{}, err
		}

		results = append(results, temp.User)
	}

	return results, nil
}

func (s *Storage) GetUsersBySearchTextCount(searchText string) (int, error) {

	var totalResultsCount int

	query := `SELECT COUNT(id) FROM users WHERE username ILIKE $1`

	searchParam := "%" + searchText + "%"

	row := s.db.QueryRow(query, searchParam)

	if err := row.Scan(&totalResultsCount); err != nil {
		return -1, err
	}

	return totalResultsCount, nil

}
