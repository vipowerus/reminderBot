package store

import (
	"database/sql"
	"strconv"

	"github.com/lib/pq"
)

// Config ...
type Config struct {
	DatabaseURL string `toml:"database_url"`
}

// NewConfig return new initialized struct
func NewConfig() *Config {
	return &Config{}
}

// Store ...
type Store struct {
	config *Config
	db     *sql.DB
}

// New ...
func New(config *Config) *Store {
	return &Store{
		config: config,
	}
}

// Open database with config data and attach DB instance to the store structure
func (s *Store) Open() error {
	db, err := sql.Open("postgres", s.config.DatabaseURL)
	if err != nil {
		return err
	}
	if err := db.Ping(); err != nil {
		return err
	}
	s.db = db
	return nil
}

// Close DB connection
func (s *Store) Close() {
	s.db.Close()
}

// AddUser with group "0" (default value) to the DB
func (s *Store) AddUser(userId int64) error {
	// @TODO Remake query
	_, err := s.db.Exec("INSERT INTO users (user_id, has_group) VALUES ($1, 0);", userId)
	return err
}

// UpdateUserHasGroup Update user's "hasGroup" field in the DB
func (s *Store) UpdateUserHasGroup(userId int64, hasGroup int8) error {
	// @TODO Remake query
	_, err := s.db.Exec("UPDATE users SET has_group = $1 WHERE user_id = $2", hasGroup, userId)
	return err
}

// UserInGroup Check if the user is in any group by his id
func (s *Store) UserInGroup(userId int64) (bool, error) {
	row := s.db.QueryRow("SELECT has_group FROM users WHERE user_id = $1;", userId)
	var hasGroup string
	if err := row.Scan(&hasGroup); err != nil {
		return false, err
	}
	boolHasGroup, _ := strconv.ParseBool(hasGroup)
	return boolHasGroup, nil
}

// AddSchedule Adds parsed schedule to the DB
func (s *Store) AddSchedule(groupNumber string, schedule [7][6]string) error {
	// @TODO Remake query
	_, err := s.db.Exec("INSERT INTO schedules (group_number, schedule) VALUES ($1, $2);", groupNumber, pq.Array(schedule))
	return err
}

// ScheduleExists ...
func (s *Store) ScheduleExists(groupNumber string) (bool, error) {
	var id int
	err := s.db.QueryRow("SELECT id FROM schedules WHERE group_number = $1;", groupNumber).Scan(&id)
	if err != nil {
		if err != sql.ErrNoRows {
			return false, err
		}
		return false, nil
	}
	return true, nil
}

// AddUserToSchedule Attaches the user to the schedule
func (s *Store) AddUserToSchedule(studentId int64, groupNumber string) error {
	// @TODO Remake query
	_, err := s.db.Exec("UPDATE schedules SET students_ids = array_append(students_ids, $1) WHERE group_number = $2",
		studentId, groupNumber)
	return err
}

// DeleteUserFromSchedule Detaches the user from the schedule
func (s *Store) DeleteUserFromSchedule(studentId int64, groupNumber string) error {
	// @TODO Remake query
	_, err := s.db.Exec("UPDATE schedules SET students_ids = array_append(students_ids, $1) WHERE group_number = $2",
		studentId, groupNumber)
	return err
}
