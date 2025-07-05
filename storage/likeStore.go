package storage

import "errors"

type Like struct {
	LikedById   int    `db:"liked_by_id" json:"liked_by_id"`
	LikedPostId int    `db:"liked_post_id" json:"liked_post_id"`
	LikedAt     string `db:"liked_at" json:"liked_at"`
}

func (s *Storage) GetLike(likedById int, likedPostId int) (*Like, error) {

	var like Like

	query := `SELECT liked_by_id,liked_post_id,liked_at 
	FROM likes WHERE liked_by_id=$1 AND liked_post_id=$2`

	if err := s.db.Get(&like, query, likedById, likedPostId); err != nil {
		return nil, err
	}

	return &like, nil
}

func (s *Storage) CreateLike(likedById int, likedPostId int) (*Like, error) {

	var like Like

	query := `INSERT INTO likes(liked_by_id,liked_post_id) VALUES($1,$2) 
	RETURNING liked_by_id,liked_post_id,liked_at`

	row := s.db.QueryRowx(query, likedById, likedPostId)

	if err := row.StructScan(&like); err != nil {
		return nil, err
	}

	return &like, nil
}

func (s *Storage) RemoveLike(likedById int, likedPostId int) error {

	query := `DELETE FROM likes WHERE liked_by_id=$1 AND liked_post_id=$2`

	result, err := s.db.Exec(query, likedById, likedPostId)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected != 1 {
		return errors.New("no of likes deleted was not one")
	}

	return nil
}

func (s *Storage) GetPostLikes(likedPostId int) ([]Like, error) {
	var likes []Like

	query := `SELECT liked_by_id,liked_post_id,liked_at FROM likes WHERE liked_post_id=$1`

	rows, err := s.db.Queryx(query, likedPostId)
	if err != nil {
		return []Like{}, err
	}

	defer rows.Close()

	for rows.Next() {
		var like Like

		if err := rows.StructScan(&like); err != nil {
			return []Like{}, err
		}

		likes = append(likes, like)
	}

	return likes, nil
}
