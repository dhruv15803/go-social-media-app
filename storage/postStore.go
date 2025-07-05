package storage

import (
	"errors"
)

type Post struct {
	Id            int     `db:"id" json:"id"`
	PostContent   string  `db:"post_content" json:"post_content"`
	UserId        int     `db:"user_id" json:"user_id"`
	ParentPostId  *int    `db:"parent_post_id" json:"parent_post_id"`
	PostCreatedAt string  `db:"post_created_at" json:"post_created_at"`
	PostUpdatedAt *string `db:"post_updated_at" json:"post_updated_at"`
}

type PostImage struct {
	Id           int    `db:"id" json:"id"`
	PostImageUrl string `db:"post_image_url" json:"post_image_url"`
	PostId       int    `db:"post_id" json:"post_id"`
}

type PostWithUser struct {
	Post
	User User `json:"user"`
}

type PostWithUserAndImages struct {
	Post
	User       User        `json:"user"`
	PostImages []PostImage `json:"post_images"`
}

type PostWithMetaData struct {
	Post
	User           User        `json:"user"`
	PostImages     []PostImage `json:"post_images"`
	LikesCount     int         `json:"likes_count"`
	CommentsCount  int         `json:"comments_count"`
	BookmarksCount int         `json:"bookmarks_count"`
}

// method for creating top-level post
func (s *Storage) CreatePost(postContent string, userId int) (*PostWithUser, error) {

	var post Post

	query := `INSERT INTO posts(post_content,user_id) VALUES($1,$2) RETURNING 
	id,post_content,user_id,parent_post_id,post_created_at,post_updated_at`

	row := s.db.QueryRowx(query, postContent, userId)

	if err := row.StructScan(&post); err != nil {
		return nil, err
	}

	var postWithUser PostWithUser
	var user User

	query = `SELECT id,email,username,image_url,password,bio,location,
	date_of_birth,is_public,created_at,updated_at FROM users WHERE id=$1`

	if err := s.db.Get(&user, query, userId); err != nil {
		return nil, err
	}

	postWithUser.Post = post
	postWithUser.User = user

	return &postWithUser, nil
}

// creating parent post with images
func (s *Storage) CreatePostWithImages(postContent string, postImageUrls []string, userId int) (*PostWithUserAndImages, error) {

	var err error
	var post Post
	var postImages []PostImage
	var user User
	var postWithUserAndImages PostWithUserAndImages

	tx, err := s.db.Beginx()
	if err != nil {
		return nil, err
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	query := `INSERT INTO posts(post_content,user_id) VALUES($1,$2) RETURNING
	id,post_content,user_id,parent_post_id,post_created_at,post_updated_at`

	row := tx.QueryRowx(query, postContent, userId)

	if err = row.StructScan(&post); err != nil {
		return nil, err
	}
	for _, postImageUrl := range postImageUrls {

		var postImage PostImage

		query = `INSERT INTO post_images(post_image_url,post_id) VALUES($1,$2)
		RETURNING id,post_image_url,post_id`

		row := tx.QueryRowx(query, postImageUrl, post.Id)

		if err = row.StructScan(&postImage); err != nil {
			return nil, err
		}

		postImages = append(postImages, postImage)
	}

	query = `SELECT id,email,username,image_url,password,
	bio,location,date_of_birth,is_public,created_at,updated_at
	FROM users WHERE id=$1`

	if err = tx.Get(&user, query, userId); err != nil {
		return nil, err
	}

	postWithUserAndImages.Post = post
	postWithUserAndImages.PostImages = postImages
	postWithUserAndImages.User = user

	tx.Commit()

	return &postWithUserAndImages, nil
}

func (s *Storage) CreateChildPost(postContent string, userId int, parentPostId int) (*PostWithUser, error) {

	var post Post
	var user User

	query := `INSERT INTO posts(post_content,user_id,parent_post_id) VALUES($1,$2,$3) 
	RETURNING id,post_content,user_id,parent_post_id,post_created_at,post_updated_at`

	row := s.db.QueryRowx(query, postContent, userId, parentPostId)

	if err := row.StructScan(&post); err != nil {
		return nil, err
	}

	var postWithUser PostWithUser

	query = `SELECT id,email,username,image_url,password,bio,location,
	date_of_birth,is_public,created_at,updated_at 
	FROM users WHERE id=$1`

	if err := s.db.Get(&user, query, userId); err != nil {
		return nil, err
	}

	postWithUser.Post = post
	postWithUser.User = user

	return &postWithUser, nil
}

func (s *Storage) CreateChildPostWithImages(postContent string, postImageUrls []string, userId int, parentPostId int) (*PostWithUserAndImages, error) {

	var post Post
	var user User
	var postImages []PostImage

	tx, err := s.db.Beginx()
	if err != nil {
		return nil, err
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	query := `INSERT INTO posts(post_content,user_id,parent_post_id) VALUES($1,$2,$3)
	RETURNING id,post_content,user_id,parent_post_id,post_created_at,post_updated_at`

	row := tx.QueryRowx(query, postContent, userId, parentPostId)

	if err := row.StructScan(&post); err != nil {
		return nil, err
	}

	for _, postImageUrl := range postImageUrls {
		var postImage PostImage
		query = `INSERT INTO post_images(post_image_url,post_id) VALUES($1,$2) RETURNING 
		id,post_image_url,post_id`
		row := tx.QueryRowx(query, postImageUrl, post.Id)
		if err := row.StructScan(&postImage); err != nil {
			return nil, err
		}
		postImages = append(postImages, postImage)
	}

	query = `SELECT id,email,username,image_url,password,bio,location,
	date_of_birth,is_public,created_at,updated_at FROM users WHERE id=$1`

	if err := tx.Get(&user, query, userId); err != nil {
		return nil, err
	}

	var postWithUserAndImages PostWithUserAndImages
	postWithUserAndImages.Post = post
	postWithUserAndImages.User = user
	postWithUserAndImages.PostImages = postImages

	tx.Commit()

	return &postWithUserAndImages, nil
}

func (s *Storage) GetPostById(id int) (*Post, error) {

	var post Post

	query := `SELECT id,post_content,user_id,parent_post_id,
	post_created_at,post_updated_at FROM posts WHERE id=$1`

	if err := s.db.Get(&post, query, id); err != nil {
		return nil, err
	}

	return &post, nil
}

func (s *Storage) GetPostWithMetaDataById(id int) (*PostWithMetaData, error) {
	var postWithMetaData PostWithMetaData

	query := `SELECT 
    	p.id,
		p.post_content,
		p.user_id,
		p.parent_post_id,
		p.post_created_at,
		p.post_updated_at,
		
		u.id,
		u.email,
		u.username,
		u.image_url,
		u.password,
		u.bio,
		u.location,
		u.date_of_birth,
		u.is_public,
		u.created_at,
		u.updated_at,
        
		COUNT(DISTINCT l.liked_by_id) AS likes_count,
        COUNT(DISTINCT c.id) AS comments_count,
        COUNT(DISTINCT b.bookmarked_by_id) AS bookmarks_count
    FROM 
        posts AS p 
        INNER JOIN users AS u ON p.user_id = u.id 
        LEFT JOIN likes AS l ON l.liked_post_id = p.id
        LEFT JOIN posts AS c ON c.parent_post_id = p.id 
        LEFT JOIN bookmarks AS b ON b.bookmarked_post_id = p.id
    WHERE 
        p.id=$1
    GROUP BY 
		p.id , u.id`

	row := s.db.QueryRowx(query, id)

	if err := row.Scan(&postWithMetaData.Id, &postWithMetaData.PostContent,
		&postWithMetaData.UserId, &postWithMetaData.ParentPostId, &postWithMetaData.PostCreatedAt,
		&postWithMetaData.PostUpdatedAt, &postWithMetaData.User.Id, &postWithMetaData.User.Email,
		&postWithMetaData.User.Username, &postWithMetaData.User.ImageUrl,
		&postWithMetaData.User.Password, &postWithMetaData.User.Bio, &postWithMetaData.User.Location,
		&postWithMetaData.User.DateOfBirth, &postWithMetaData.User.IsPublic, &postWithMetaData.User.CreatedAt,
		&postWithMetaData.User.UpdatedAt, &postWithMetaData.LikesCount, &postWithMetaData.CommentsCount,
		&postWithMetaData.BookmarksCount); err != nil {
		return nil, err
	}

	var postImages []PostImage

	imageQuery := `SELECT id,post_image_url,post_id FROM post_images WHERE post_id=$1`

	imageRows, err := s.db.Queryx(imageQuery, postWithMetaData.Id)
	if err != nil {
		return nil, err
	}

	defer imageRows.Close()

	for imageRows.Next() {
		var postImage PostImage

		if err := imageRows.StructScan(&postImage); err != nil {
			return nil, err
		}

		postImages = append(postImages, postImage)
	}

	postWithMetaData.PostImages = postImages

	return &postWithMetaData, nil
}

func (s *Storage) DeletePostById(id int) error {

	query := `DELETE FROM posts WHERE id=$1`

	result, err := s.db.Exec(query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected != 1 {
		return errors.New("no of posts deleted is not 1")
	}

	return nil
}

func (s *Storage) GetUserPostFeed(skip int, limit int, userId int, likesCountWt, commentsCountWt, bookmarksCountWt float64) ([]PostWithMetaData, error) {

	// the userId is the logged in user id , so getting post's feed
	// for user  .
	// -> public posts , posts of the user's this user follows , its own posts

	var postsWithMetaData []PostWithMetaData

	query := `SELECT * , $4::numeric * q.likes_count + $5::numeric * q.comments_count + $6::numeric * q.bookmarks_count AS activity_score  
	FROM (
	SELECT 
    	p.id,
		p.post_content,
		p.user_id,
		p.parent_post_id,
		p.post_created_at,
		p.post_updated_at,
		
		u.id,
		u.email,
		u.username,
		u.image_url,
		u.password,
		u.bio,
		u.location,
		u.date_of_birth,
		u.is_public,
		u.created_at,
		u.updated_at,
        
		COUNT(DISTINCT l.liked_by_id) AS likes_count,
        COUNT(DISTINCT c.id) AS comments_count,
        COUNT(DISTINCT b.bookmarked_by_id) AS bookmarks_count
    FROM 
        posts AS p 
        INNER JOIN users AS u ON p.user_id = u.id 
        LEFT JOIN likes AS l ON l.liked_post_id = p.id
        LEFT JOIN posts AS c ON c.parent_post_id = p.id 
        LEFT JOIN bookmarks AS b ON b.bookmarked_post_id = p.id
    WHERE 
        p.parent_post_id IS NULL AND (u.is_public=true OR u.id IN (SELECT following_id FROM follows WHERE follower_id=$3) OR u.id=$3)
    GROUP BY 
		p.id , u.id
) AS q 
	ORDER BY activity_score DESC , post_created_at DESC
    LIMIT $1 OFFSET $2`

	rows, err := s.db.Queryx(query, limit, skip, userId, likesCountWt, commentsCountWt, bookmarksCountWt)
	if err != nil {
		return []PostWithMetaData{}, err
	}

	defer rows.Close()

	for rows.Next() {

		var postWithMetaData PostWithMetaData
		var activityStore float64

		if err := rows.Scan(&postWithMetaData.Id, &postWithMetaData.PostContent, &postWithMetaData.UserId, &postWithMetaData.ParentPostId, &postWithMetaData.PostCreatedAt, &postWithMetaData.PostUpdatedAt,
			&postWithMetaData.User.Id, &postWithMetaData.User.Email, &postWithMetaData.User.Username, &postWithMetaData.User.ImageUrl, &postWithMetaData.User.Password, &postWithMetaData.User.Bio,
			&postWithMetaData.User.Location, &postWithMetaData.User.DateOfBirth, &postWithMetaData.User.IsPublic, &postWithMetaData.User.CreatedAt, &postWithMetaData.User.UpdatedAt,
			&postWithMetaData.LikesCount, &postWithMetaData.CommentsCount, &postWithMetaData.BookmarksCount, &activityStore); err != nil {
			return []PostWithMetaData{}, err
		}

		// each post can have multiple images
		var postImages []PostImage

		query = `SELECT id,post_image_url,post_id 
		FROM post_images WHERE post_id=$1`

		imageRows, err := s.db.Queryx(query, postWithMetaData.Id)
		if err != nil {
			return []PostWithMetaData{}, err
		}

		defer imageRows.Next()

		for imageRows.Next() {
			var postImage PostImage
			if err := imageRows.StructScan(&postImage); err != nil {
				return []PostWithMetaData{}, err
			}
			postImages = append(postImages, postImage)
		}

		postWithMetaData.PostImages = postImages
		postsWithMetaData = append(postsWithMetaData, postWithMetaData)
	}

	return postsWithMetaData, nil
}

func (s *Storage) GetUserPostFeedCount(userId int) (int, error) {

	var userPostFeedCount int

	query := `SELECT COUNT(*) FROM 
	posts AS p INNER JOIN users AS u 
	ON p.user_id = u.id  
	WHERE p.parent_post_id IS NULL AND (u.is_public=true OR u.id IN (SELECT following_id FROM follows WHERE follower_id=$1) OR u.id = $1)
	`

	row := s.db.QueryRow(query, userId)

	if err := row.Scan(&userPostFeedCount); err != nil {
		return -1, err
	}

	return userPostFeedCount, nil

}

func (s *Storage) GetPublicPosts(skip int, limit int, likesCountWt, commentsCountWt, bookmarksCountWt float64) ([]PostWithMetaData, error) {

	var postsWithMetaData []PostWithMetaData

	query := `SELECT *,
	$3::numeric * q.likes_count + $4::numeric * q.comments_count +  $5::numeric * q.bookmarks_count AS activity_score
	FROM (
    SELECT 
    	p.id,
		p.post_content,
		p.user_id,
		p.parent_post_id,
		p.post_created_at,
		p.post_updated_at,
		
		u.id,
		u.email,
		u.username,
		u.image_url,
		u.password,
		u.bio,
		u.location,
		u.date_of_birth,
		u.is_public,
		u.created_at,
		u.updated_at,
        
		COUNT(DISTINCT l.liked_by_id) AS likes_count,
        COUNT(DISTINCT c.id) AS comments_count,
        COUNT(DISTINCT b.bookmarked_by_id) AS bookmarks_count
    FROM 
        posts AS p 
        INNER JOIN users AS u ON p.user_id = u.id 
        LEFT JOIN likes AS l ON l.liked_post_id = p.id
        LEFT JOIN posts AS c ON c.parent_post_id = p.id 
        LEFT JOIN bookmarks AS b ON b.bookmarked_post_id = p.id
    WHERE 
        p.parent_post_id IS NULL AND u.is_public=true
    GROUP BY 
        p.id , u.id
) AS q
	ORDER BY activity_score DESC , post_created_at DESC
	LIMIT $1 OFFSET $2`

	rows, err := s.db.Queryx(query, limit, skip, likesCountWt, commentsCountWt, bookmarksCountWt)
	if err != nil {
		return []PostWithMetaData{}, err
	}

	defer rows.Close()

	for rows.Next() {

		var postWithMetaData PostWithMetaData
		var activityStore float64

		if err := rows.Scan(&postWithMetaData.Id, &postWithMetaData.PostContent, &postWithMetaData.UserId, &postWithMetaData.ParentPostId, &postWithMetaData.PostCreatedAt, &postWithMetaData.PostUpdatedAt,
			&postWithMetaData.User.Id, &postWithMetaData.User.Email, &postWithMetaData.User.Username, &postWithMetaData.User.ImageUrl, &postWithMetaData.User.Password, &postWithMetaData.User.Bio,
			&postWithMetaData.User.Location, &postWithMetaData.User.DateOfBirth, &postWithMetaData.User.IsPublic, &postWithMetaData.User.CreatedAt, &postWithMetaData.User.UpdatedAt,
			&postWithMetaData.LikesCount, &postWithMetaData.CommentsCount, &postWithMetaData.BookmarksCount, &activityStore); err != nil {
			return []PostWithMetaData{}, err
		}

		// each post can have multiple images
		var postImages []PostImage

		query = `SELECT id,post_image_url,post_id 
		FROM post_images WHERE post_id=$1`

		imageRows, err := s.db.Queryx(query, postWithMetaData.Id)
		if err != nil {
			return []PostWithMetaData{}, err
		}

		defer imageRows.Next()

		for imageRows.Next() {
			var postImage PostImage
			if err := imageRows.StructScan(&postImage); err != nil {
				return []PostWithMetaData{}, err
			}
			postImages = append(postImages, postImage)
		}

		postWithMetaData.PostImages = postImages
		postsWithMetaData = append(postsWithMetaData, postWithMetaData)
	}

	return postsWithMetaData, nil
}

// parent posts  (top-level) posts count
func (s *Storage) GetPublicPostsCount() (int, error) {

	var topLevelPublicPostsCount int

	query := `SELECT COUNT(*) 
	FROM posts AS p
	INNER JOIN users AS u 
	ON p.user_id=u.id
	WHERE p.parent_post_id IS NULL AND u.is_public=true`

	row := s.db.QueryRowx(query)

	if err := row.Scan(&topLevelPublicPostsCount); err != nil {
		return -1, err
	}

	return topLevelPublicPostsCount, nil
}

func (s *Storage) GetPostsByUserId(userId int, skip int, limit int) ([]PostWithMetaData, error) {

	var postsWithMetaData []PostWithMetaData

	query := `SELECT 
        p.id,
		p.post_content,
		p.user_id,
		p.parent_post_id,
		p.post_created_at,
		p.post_updated_at,
	
		u.id,
		u.email,
		u.username,
		u.image_url,
		u.password,
		u.bio,
		u.location,
		u.date_of_birth,
		u.is_public,
		u.created_at,
		u.updated_at,
    
		COUNT(DISTINCT l.liked_by_id) AS likes_count,
        COUNT(DISTINCT c.id) AS comments_count,
        COUNT(DISTINCT b.bookmarked_by_id) AS bookmarks_count
    FROM 
        posts AS p 
        INNER JOIN users AS u ON p.user_id = u.id 
        LEFT JOIN likes AS l ON l.liked_post_id = p.id
        LEFT JOIN posts AS c ON c.parent_post_id = p.id 
        LEFT JOIN bookmarks AS b ON b.bookmarked_post_id = p.id
    WHERE 
        p.parent_post_id IS NULL AND p.user_id=$1
    GROUP BY 
        p.id , u.id
	ORDER BY p.post_created_at DESC
	OFFSET $2 LIMIT $3`

	rows, err := s.db.Queryx(query, userId, skip, limit)
	if err != nil {
		return []PostWithMetaData{}, err
	}
	defer rows.Close()

	for rows.Next() {

		var postWithMetaData PostWithMetaData

		if err := rows.Scan(&postWithMetaData.Id, &postWithMetaData.PostContent, &postWithMetaData.UserId,
			&postWithMetaData.ParentPostId, &postWithMetaData.PostCreatedAt, &postWithMetaData.PostUpdatedAt, &postWithMetaData.User.Id,
			&postWithMetaData.User.Email, &postWithMetaData.User.Username, &postWithMetaData.User.ImageUrl, &postWithMetaData.User.Password, &postWithMetaData.User.Bio,
			&postWithMetaData.User.Location, &postWithMetaData.User.DateOfBirth, &postWithMetaData.User.IsPublic, &postWithMetaData.User.CreatedAt, &postWithMetaData.User.UpdatedAt,
			&postWithMetaData.LikesCount, &postWithMetaData.CommentsCount, &postWithMetaData.BookmarksCount); err != nil {
			return []PostWithMetaData{}, err
		}

		var postImages []PostImage

		query := `SELECT id,post_image_url,post_id FROM post_images WHERE post_id=$1`

		imageRows, err := s.db.Queryx(query, postWithMetaData.Id)
		if err != nil {
			return []PostWithMetaData{}, err
		}

		defer imageRows.Close()

		for imageRows.Next() {

			var postImage PostImage

			if err := imageRows.StructScan(&postImage); err != nil {
				return []PostWithMetaData{}, err
			}

			postImages = append(postImages, postImage)
		}

		postWithMetaData.PostImages = postImages
		postsWithMetaData = append(postsWithMetaData, postWithMetaData)
	}

	return postsWithMetaData, nil
}

func (s *Storage) GetPostsCountByUser(userId int) (int, error) {

	var usersTopLevelPostsCount int

	query := `SELECT COUNT(*) FROM posts WHERE parent_post_id IS NULL AND user_id=$1`

	row := s.db.QueryRow(query, userId)

	if err := row.Scan(&usersTopLevelPostsCount); err != nil {
		return -1, err
	}

	return usersTopLevelPostsCount, nil
}

// child posts for post -> parent post
func (s *Storage) GetPostComments(postId int, skip int, limit int) ([]PostWithMetaData, error) {

	var postsWithMetaData []PostWithMetaData

	query := `SELECT 
        p.id,
		p.post_content,
		p.user_id,
		p.parent_post_id,
		p.post_created_at,
		p.post_updated_at,
	
		u.id,
		u.email,
		u.username,
		u.image_url,
		u.password,
		u.bio,
		u.location,
		u.date_of_birth,
		u.is_public,
		u.created_at,
		u.updated_at,
    
		COUNT(DISTINCT l.liked_by_id) AS likes_count,
        COUNT(DISTINCT c.id) AS comments_count,
        COUNT(DISTINCT b.bookmarked_by_id) AS bookmarks_count
    FROM 
        posts AS p 
        INNER JOIN users AS u ON p.user_id = u.id 
        LEFT JOIN likes AS l ON l.liked_post_id = p.id
        LEFT JOIN posts AS c ON c.parent_post_id = p.id 
        LEFT JOIN bookmarks AS b ON b.bookmarked_post_id = p.id
    WHERE 
        p.parent_post_id=$1
    GROUP BY 
        p.id , u.id
	ORDER BY p.post_created_at DESC
	OFFSET $2 LIMIT $3`

	rows, err := s.db.Queryx(query, postId, skip, limit)
	if err != nil {
		return []PostWithMetaData{}, err
	}

	defer rows.Close()

	for rows.Next() {

		var postWithMetaData PostWithMetaData

		if err := rows.Scan(&postWithMetaData.Id, &postWithMetaData.PostContent, &postWithMetaData.UserId,
			&postWithMetaData.ParentPostId, &postWithMetaData.PostCreatedAt, &postWithMetaData.PostUpdatedAt, &postWithMetaData.User.Id,
			&postWithMetaData.User.Email, &postWithMetaData.User.Username, &postWithMetaData.User.ImageUrl, &postWithMetaData.User.Password, &postWithMetaData.User.Bio,
			&postWithMetaData.User.Location, &postWithMetaData.User.DateOfBirth, &postWithMetaData.User.IsPublic, &postWithMetaData.User.CreatedAt, &postWithMetaData.User.UpdatedAt,
			&postWithMetaData.LikesCount, &postWithMetaData.CommentsCount, &postWithMetaData.BookmarksCount); err != nil {
			return []PostWithMetaData{}, err
		}

		var postImages []PostImage

		query = `SELECT id,post_image_url,post_id FROM post_images WHERE post_id=$1`

		imageRows, err := s.db.Queryx(query, postWithMetaData.Id)
		if err != nil {
			return []PostWithMetaData{}, err
		}

		defer imageRows.Close()

		for imageRows.Next() {
			var postImage PostImage

			if err := imageRows.StructScan(&postImage); err != nil {
				return []PostWithMetaData{}, err
			}

			postImages = append(postImages, postImage)
		}

		postWithMetaData.PostImages = postImages
		postsWithMetaData = append(postsWithMetaData, postWithMetaData)
	}

	return postsWithMetaData, nil
}

func (s *Storage) GetPostCommentsCount(postId int) (int, error) {
	var totalCommentsCountForPost int

	query := `SELECT COUNT(*) FROM posts WHERE parent_post_id=$1`

	row := s.db.QueryRow(query, postId)

	if err := row.Scan(&totalCommentsCountForPost); err != nil {
		return -1, err
	}

	return totalCommentsCountForPost, nil
}

func (s *Storage) GetLikedPostsByUser(userId int, skip int, limit int) ([]PostWithMetaData, error) {
	var postsWithMetaData []PostWithMetaData

	query := `SELECT 
        p.id,
		p.post_content,
		p.user_id,
		p.parent_post_id,
		p.post_created_at,
		p.post_updated_at,
	
		u.id,
		u.email,
		u.username,
		u.image_url,
		u.password,
		u.bio,
		u.location,
		u.date_of_birth,
		u.is_public,
		u.created_at,
		u.updated_at,
    
		COUNT(DISTINCT l.liked_by_id) AS likes_count,
        COUNT(DISTINCT c.id) AS comments_count,
        COUNT(DISTINCT b.bookmarked_by_id) AS bookmarks_count
    FROM 
        posts AS p 
        INNER JOIN users AS u ON p.user_id = u.id 
        LEFT JOIN likes AS l ON l.liked_post_id = p.id
        LEFT JOIN posts AS c ON c.parent_post_id = p.id 
        LEFT JOIN bookmarks AS b ON b.bookmarked_post_id = p.id
    WHERE 
        p.id IN (SELECT liked_post_id FROM likes WHERE liked_by_id=$1 ORDER BY liked_at DESC)
    GROUP BY 
        p.id , u.id
	OFFSET $2 LIMIT $3`

	rows, err := s.db.Queryx(query, userId, skip, limit)
	if err != nil {
		return []PostWithMetaData{}, err
	}

	defer rows.Close()

	for rows.Next() {

		var postWithMetaData PostWithMetaData

		if err := rows.Scan(&postWithMetaData.Id, &postWithMetaData.PostContent, &postWithMetaData.UserId,
			&postWithMetaData.ParentPostId, &postWithMetaData.PostCreatedAt, &postWithMetaData.PostUpdatedAt, &postWithMetaData.User.Id,
			&postWithMetaData.User.Email, &postWithMetaData.User.Username, &postWithMetaData.User.ImageUrl, &postWithMetaData.User.Password, &postWithMetaData.User.Bio,
			&postWithMetaData.User.Location, &postWithMetaData.User.DateOfBirth, &postWithMetaData.User.IsPublic, &postWithMetaData.User.CreatedAt, &postWithMetaData.User.UpdatedAt,
			&postWithMetaData.LikesCount, &postWithMetaData.CommentsCount, &postWithMetaData.BookmarksCount); err != nil {
			return []PostWithMetaData{}, err
		}

		var postImages []PostImage

		query = `SELECT id,post_image_url,post_id FROM post_images WHERE post_id=$1`

		imageRows, err := s.db.Queryx(query, postWithMetaData.Id)
		if err != nil {
			return []PostWithMetaData{}, err
		}

		defer imageRows.Close()

		for imageRows.Next() {
			var postImage PostImage

			if err := imageRows.StructScan(&postImage); err != nil {
				return []PostWithMetaData{}, err
			}

			postImages = append(postImages, postImage)
		}

		postWithMetaData.PostImages = postImages

		postsWithMetaData = append(postsWithMetaData, postWithMetaData)
	}

	return postsWithMetaData, nil
}

func (s *Storage) GetLikedPostsByUserCount(userId int) (int, error) {
	var likedPostsByUserCount int

	query := `SELECT COUNT(liked_post_id) FROM likes WHERE liked_by_id=$1`

	row := s.db.QueryRow(query, userId)

	if err := row.Scan(&likedPostsByUserCount); err != nil {
		return -1, err
	}

	return likedPostsByUserCount, nil
}

func (s *Storage) GetBookmarkedPostsByUser(userId int, skip int, limit int) ([]PostWithMetaData, error) {
	var postsWithMetaData []PostWithMetaData

	query := `SELECT 
        p.id,
		p.post_content,
		p.user_id,
		p.parent_post_id,
		p.post_created_at,
		p.post_updated_at,
	
		u.id,
		u.email,
		u.username,
		u.image_url,
		u.password,
		u.bio,
		u.location,
		u.date_of_birth,
		u.is_public,
		u.created_at,
		u.updated_at,
    
		COUNT(DISTINCT l.liked_by_id) AS likes_count,
        COUNT(DISTINCT c.id) AS comments_count,
        COUNT(DISTINCT b.bookmarked_by_id) AS bookmarks_count
    FROM 
        posts AS p 
        INNER JOIN users AS u ON p.user_id = u.id 
        LEFT JOIN likes AS l ON l.liked_post_id = p.id
        LEFT JOIN posts AS c ON c.parent_post_id = p.id 
        LEFT JOIN bookmarks AS b ON b.bookmarked_post_id = p.id
    WHERE 
        p.id IN (SELECT bookmarked_post_id FROM bookmarks WHERE bookmarked_by_id=$1 ORDER BY bookmarked_at)
    GROUP BY 
        p.id , u.id
	OFFSET $2 LIMIT $3`

	rows, err := s.db.Queryx(query, userId, skip, limit)
	if err != nil {
		return []PostWithMetaData{}, err
	}

	defer rows.Close()

	for rows.Next() {

		var postWithMetaData PostWithMetaData

		if err := rows.Scan(&postWithMetaData.Id, &postWithMetaData.PostContent, &postWithMetaData.UserId,
			&postWithMetaData.ParentPostId, &postWithMetaData.PostCreatedAt, &postWithMetaData.PostUpdatedAt, &postWithMetaData.User.Id,
			&postWithMetaData.User.Email, &postWithMetaData.User.Username, &postWithMetaData.User.ImageUrl, &postWithMetaData.User.Password, &postWithMetaData.User.Bio,
			&postWithMetaData.User.Location, &postWithMetaData.User.DateOfBirth, &postWithMetaData.User.IsPublic, &postWithMetaData.User.CreatedAt, &postWithMetaData.User.UpdatedAt,
			&postWithMetaData.LikesCount, &postWithMetaData.CommentsCount, &postWithMetaData.BookmarksCount); err != nil {
			return []PostWithMetaData{}, err
		}

		var postImages []PostImage

		query = `SELECT id,post_image_url,post_id FROM post_images WHERE post_id=$1`

		imageRows, err := s.db.Queryx(query, postWithMetaData.Id)
		if err != nil {
			return []PostWithMetaData{}, err
		}

		defer imageRows.Close()

		for imageRows.Next() {
			var postImage PostImage

			if err := imageRows.StructScan(&postImage); err != nil {
				return []PostWithMetaData{}, err
			}

			postImages = append(postImages, postImage)
		}

		postWithMetaData.PostImages = postImages

		postsWithMetaData = append(postsWithMetaData, postWithMetaData)
	}

	return postsWithMetaData, nil

}

func (s *Storage) GetBookmarkedPostsByUserCount(userId int) (int, error) {

	var totalBookmarkedPostsCount int

	query := `SELECT COUNT(bookmarked_post_id) FROM bookmarks WHERE bookmarked_by_id=$1`

	if err := s.db.Get(&totalBookmarkedPostsCount, query, userId); err != nil {
		return -1, err
	}

	return totalBookmarkedPostsCount, nil

}
