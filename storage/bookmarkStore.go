package storage

import "errors"

type Bookmark struct {
	BookmarkedById   int    `db:"bookmarked_by_id" json:"bookmarked_by_id"`
	BookmarkedPostId int    `db:"bookmarked_post_id" json:"bookmarked_post_id"`
	BookmarkedAt     string `db:"bookmarked_at" json:"bookmarked_at"`
}

func (s *Storage) CreateBookmark(bookmarkedById int, bookmarkedPostId int) (*Bookmark, error) {
	var bookmark Bookmark

	query := `INSERT INTO bookmarks(bookmarked_by_id,bookmarked_post_id) VALUES($1,$2) 
	RETURNING bookmarked_by_id,bookmarked_post_id,bookmarked_at`

	row := s.db.QueryRowx(query, bookmarkedById, bookmarkedPostId)

	if err := row.StructScan(&bookmark); err != nil {
		return nil, err
	}

	return &bookmark, nil

}

func (s *Storage) RemoveBookmark(bookmarkedById int, bookmarkedPostId int) error {

	query := `DELETE FROM bookmarks WHERE bookmarked_by_id=$1 AND bookmarked_post_id=$2`

	result, err := s.db.Exec(query, bookmarkedById, bookmarkedPostId)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected != 1 {
		return errors.New("no of bookmarks deleted not one")
	}

	return nil
}

func (s *Storage) GetBookmark(bookmarkedById int, bookmarkedPostId int) (*Bookmark, error) {

	var bookmark Bookmark

	query := `SELECT bookmarked_by_id,bookmarked_post_id,bookmarked_at FROM 
	bookmarks WHERE bookmarked_by_id=$1 AND bookmarked_post_id=$2`

	if err := s.db.Get(&bookmark, query, bookmarkedById, bookmarkedPostId); err != nil {
		return nil, err
	}

	return &bookmark, nil

}

func (s *Storage) GetBookmarksByPostId(postId int) ([]Bookmark, error) {
	var bookmarks []Bookmark

	query := `SELECT bookmarked_by_id,bookmarked_post_id,bookmarked_at FROM 
	bookmarks WHERE bookmarked_post_id=$1`

	rows, err := s.db.Queryx(query, postId)
	if err != nil {
		return []Bookmark{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var bookmark Bookmark

		if err := rows.StructScan(&bookmark); err != nil {
			return []Bookmark{}, err
		}
		bookmarks = append(bookmarks, bookmark)
	}

	return bookmarks, nil
}
