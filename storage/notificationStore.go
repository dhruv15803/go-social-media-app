package storage

type NotificationType string

type Notification struct {
	Id                    int              `db:"id" json:"id"`
	UserId                int              `db:"user_id" json:"user_id"`
	NotificationType      NotificationType `db:"notification_type" json:"notification_type"`
	ActorId               int              `db:"actor_id" json:"actor_id"`
	NotificationCreatedAt string           `db:"notification_created_at" json:"notification_created_at"`
	PostId                int              `db:"post_id" json:"post_id"`
}

type NotificationWithActor struct {
	Notification
	Actor User `json:"actor"`
}

func (s *Storage) CreateNotification(userId int, actorId int, postId int, notificationType NotificationType) (*Notification, error) {

	var notification Notification

	query := `INSERT INTO notifications(user_id,notification_type,actor_id,post_id) VALUES($1,$2,$3,$4) 
	RETURNING id,user_id,notification_type,actor_id,notification_created_at,post_id`

	row := s.db.QueryRowx(query, userId, notificationType, actorId, postId)

	if err := row.StructScan(&notification); err != nil {
		return nil, err
	}

	return &notification, nil
}

func (s *Storage) GetNotificationsByUserId(userId int, skip int, limit int) ([]NotificationWithActor, error) {
	var notifications []NotificationWithActor

	query := `SELECT n.id,n.user_id,n.notification_type,n.actor_id,n.notification_created_at,n.post_id,u.id,
u.email, u.username,u.image_url,u.password,u.bio,u.location,u.date_of_birth,u.is_public,u.created_at, 
u.updated_at 
FROM 
	notifications AS n INNER JOIN users AS u ON n.actor_id=u.id
WHERE 
	n.user_id=$1 
ORDER BY 
	n.notification_created_at DESC 
LIMIT $2 OFFSET $3`

	rows, err := s.db.Queryx(query, userId, limit, skip)
	if err != nil {
		return []NotificationWithActor{}, err
	}

	defer rows.Close()

	for rows.Next() {
		var notification NotificationWithActor

		if err := rows.Scan(&notification.Id, &notification.UserId, &notification.NotificationType, &notification.ActorId,
			&notification.NotificationCreatedAt, &notification.PostId, &notification.Actor.Id, &notification.Actor.Email, &notification.Actor.Username,
			&notification.Actor.ImageUrl, &notification.Actor.Password, &notification.Actor.Bio, &notification.Actor.Location,
			&notification.Actor.DateOfBirth, &notification.Actor.IsPublic, &notification.Actor.CreatedAt,
			&notification.Actor.UpdatedAt); err != nil {
			return []NotificationWithActor{}, err
		}

		notifications = append(notifications, notification)

	}

	return notifications, nil
}

func (s *Storage) GetNotificationsByUserIdCount(userId int) (int, error) {

	var totalNotificationsCount int

	query := `SELECT COUNT(id) FROM notifications WHERE user_id=$1`

	if err := s.db.Get(&totalNotificationsCount, query, userId); err != nil {
		return -1, err
	}

	return totalNotificationsCount, nil
}

func (s *Storage) GetNotificationsByActorIdAndPostId(actorId int, postId int, notificationType NotificationType) ([]Notification, error) {

	var notifications []Notification

	query := `SELECT id,user_id,notification_type,actor_id,notification_created_at,post_id FROM notifications WHERE actor_id=$1 AND post_id=$2 AND notification_type=$3`

	rows, err := s.db.Queryx(query, actorId, postId, notificationType)
	if err != nil {
		return []Notification{}, err
	}

	defer rows.Close()

	for rows.Next() {

		var notification Notification

		if err := rows.StructScan(&notification); err != nil {
			return []Notification{}, err

		}

		notifications = append(notifications, notification)
	}

	return notifications, nil
}

func (s *Storage) UpdateNotificationByActorIdAndPostId(actorId int, postId int, notificationType NotificationType) (*Notification, error) {

	var updatedNotification Notification

	query := `UPDATE notifications
		SET notification_created_at=now()
		WHERE actor_id=$1 AND post_id=$2 AND notification_type=$3 
		RETURNING id,user_id,notification_type,actor_id,notification_created_at,post_id
		`

	if err := s.db.Get(&updatedNotification, query, actorId, postId, notificationType); err != nil {
		return nil, err
	}

	return &updatedNotification, nil
}
