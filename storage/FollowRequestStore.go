package storage

import "errors"

type FollowRequest struct {
	RequestSenderId   int    `db:"request_sender_id" json:"request_sender_id"`
	RequestReceiverId int    `db:"request_receiver_id" json:"request_receiver_id"`
	RequestAt         string `db:"request_at" json:"request_at"`
}

type FollowRequestWithSender struct {
	FollowRequest
	RequestSender User `json:"request_sender"`
}

func (s *Storage) CreateFollowRequest(requestSenderId int, requestReceiverId int) (*FollowRequest, error) {
	var followRequest FollowRequest

	query := `INSERT INTO follow_requests(request_sender_id,request_receiver_id) VALUES($1,$2) 
	RETURNING request_sender_id,request_receiver_id,request_at`

	row := s.db.QueryRowx(query, requestSenderId, requestReceiverId)

	if err := row.StructScan(&followRequest); err != nil {
		return nil, err
	}

	return &followRequest, nil
}

func (s *Storage) RemoveFollowRequest(requestSenderId int, requestReceiverId int) error {

	query := `DELETE FROM follow_requests WHERE request_sender_id=$1 AND request_receiver_id=$2`

	result, err := s.db.Exec(query, requestSenderId, requestReceiverId)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected != 1 {
		return errors.New("no of follow request deleted is not one")
	}

	return nil
}

func (s *Storage) GetFollowRequest(requestSenderId int, requestReceiverId int) (*FollowRequest, error) {

	var followRequest FollowRequest

	query := `SELECT request_sender_id,request_receiver_id,request_at FROM 
	follow_requests WHERE request_sender_id=$1 AND request_receiver_id=$2`

	if err := s.db.Get(&followRequest, query, requestSenderId, requestReceiverId); err != nil {
		return nil, err
	}
	return &followRequest, nil
}

func (s *Storage) AcceptFollowRequest(requestSenderId int, requestReceiverId int) (*Follow, error) {

	var follow Follow

	query := `INSERT INTO follows(follower_id,following_id) VALUES($1,$2) 
	RETURNING follower_id,following_id,followed_at`

	tx, err := s.db.Beginx()
	if err != nil {
		return nil, err
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	row := tx.QueryRowx(query, requestSenderId, requestReceiverId)

	if err := row.StructScan(&follow); err != nil {
		return nil, err
	}

	// after follow is created , there should not be a request that exists

	query = `DELETE FROM follow_requests WHERE request_sender_id=$1 AND request_receiver_id=$2`

	result, err := tx.Exec(query, requestSenderId, requestReceiverId)
	if err != nil {
		return nil, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}

	if rowsAffected != 1 {
		return nil, errors.New("no of requests deleted is not one")
	}

	tx.Commit()

	return &follow, nil
}

func (s *Storage) GetFollowRequestsSentByUser(userId int) ([]FollowRequest, error) {

	var followRequests []FollowRequest

	query := `SELECT request_sender_id,request_receiver_id,request_at FROM follow_requests WHERE request_sender_id=$1`

	rows, err := s.db.Queryx(query, userId)
	if err != nil {
		return []FollowRequest{}, err
	}

	defer rows.Close()

	for rows.Next() {

		var followRequest FollowRequest

		if err := rows.StructScan(&followRequest); err != nil {
			return []FollowRequest{}, err
		}

		followRequests = append(followRequests, followRequest)
	}

	return followRequests, nil
}

func (s *Storage) GetFollowRequestsReceivedByUser(userId int, skip int, limit int) ([]FollowRequestWithSender, error) {
	var followRequests []FollowRequestWithSender

	query := `SELECT fr.request_sender_id,fr.request_receiver_id,fr.request_at,
u.id,u.email,u.username,u.image_url,u.password,u.bio,u.location,u.date_of_birth,u.is_public,
u.created_at,u.updated_at
FROM follow_requests AS fr INNER JOIN users AS u ON fr.request_sender_id=u.id 
WHERE request_receiver_id=$1
ORDER BY request_at DESC
LIMIT $2 OFFSET $3`

	rows, err := s.db.Queryx(query, userId, limit, skip)
	if err != nil {
		return []FollowRequestWithSender{}, err
	}

	defer rows.Close()

	for rows.Next() {

		var followRequest FollowRequestWithSender

		if err := rows.Scan(&followRequest.RequestSenderId, &followRequest.RequestReceiverId, &followRequest.RequestAt, &followRequest.RequestSender.Id,
			&followRequest.RequestSender.Email, &followRequest.RequestSender.Username, &followRequest.RequestSender.ImageUrl, &followRequest.RequestSender.Password,
			&followRequest.RequestSender.Bio, &followRequest.RequestSender.Location, &followRequest.RequestSender.DateOfBirth, &followRequest.RequestSender.IsPublic, &followRequest.RequestSender.CreatedAt, &followRequest.RequestSender.UpdatedAt); err != nil {
			return []FollowRequestWithSender{}, err
		}

		followRequests = append(followRequests, followRequest)

	}

	return followRequests, nil
}

func (s *Storage) GetFollowRequestsReceivedByUserCount(userId int) (int, error) {

	var totalRequestsCount int

	query := `SELECT COUNT(*) FROM follow_requests WHERE request_receiver_id=$1`

	if err := s.db.Get(&totalRequestsCount, query, userId); err != nil {
		return -1, err
	}

	return totalRequestsCount, nil

}
