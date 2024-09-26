package storage

import (
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type PostgresDB struct {
	*gorm.DB
}

func NewPostgresDB(host, user, password, dbname string, port int) (*PostgresDB, error) {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=disable TimeZone=Asia/Taipei",
		host, user, password, dbname, port)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}

	return &PostgresDB{DB: db}, nil
}

func (db *PostgresDB) Close() error {
	sqlDB, err := db.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// AutoMigrate 自動遷移資料庫結構
func (db *PostgresDB) AutoMigrate(models ...interface{}) error {
	return db.DB.AutoMigrate(models...)
}
