package storage

type NotificationType string

type Notification struct {
	Id                    int              `db:"id" json:"id"`
	UserId                int              `db:"user_id" json:"user_id"`
	NotificationType      NotificationType `db:"notification_type" json:"notification_type"`
	ActorId               int              `db:"actor_id" json:"actor_id"`
	NotificationCreatedAt string           `db:"notification_created_at" json:"notification_created_at"`
}

func (s *Storage) CreateNotification(userId int, actorId int, notificationType NotificationType) (*Notification, error) {

	var notification Notification

	query := `INSERT INTO notifications(user_id,notification_type,actor_id) VALUES($1,$2,$3) 
	RETURNING id,user_id,notification_type,actor_id,notification_created_at`

	row := s.db.QueryRowx(query, userId, notificationType, actorId)

	if err := row.StructScan(&notification); err != nil {
		return nil, err
	}

	return &notification, nil
}
