package repository

import "debate_web/internal/storage"

type BaseRepository interface {
	Create(model interface{}) error
	FindByID(id uint, model interface{}) error
	Update(model interface{}) error
	Delete(model interface{}) error
}

type baseRepository struct {
	db *storage.PostgresDB
}

func NewBaseRepository(db *storage.PostgresDB) BaseRepository {
	return &baseRepository{db: db}
}

func (r *baseRepository) Create(model interface{}) error {
	return r.db.Create(model).Error
}

func (r *baseRepository) FindByID(id uint, model interface{}) error {
	return r.db.First(model, id).Error
}

func (r *baseRepository) Update(model interface{}) error {
	return r.db.Save(model).Error
}

func (r *baseRepository) Delete(model interface{}) error {
	return r.db.Delete(model).Error
}
