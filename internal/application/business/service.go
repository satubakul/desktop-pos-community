package business

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	domain "github.com/satubakul/desktop-pos-community/internal/domain/business"
)

type Repository interface {
	Create(context.Context, domain.Business) error
	List(context.Context) ([]domain.Business, error)
	ActiveID(context.Context) (string, error)
	SetActive(context.Context, string) error
}

type CreateInput struct {
	Name       string
	Phone      string
	Address    string
	OutletName string
	OutletAddr string
	Profiles   []domain.Profile
}

type Service struct {
	repository Repository
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (domain.Business, error) {
	businessID, err := newID()
	if err != nil {
		return domain.Business{}, err
	}
	outletID, err := newID()
	if err != nil {
		return domain.Business{}, err
	}

	now := time.Now().UTC()
	created := domain.Business{
		ID:        businessID,
		Name:      strings.TrimSpace(input.Name),
		Phone:     strings.TrimSpace(input.Phone),
		Address:   strings.TrimSpace(input.Address),
		CreatedAt: now,
		Profiles:  input.Profiles,
		Outlets: []domain.Outlet{{
			ID:         outletID,
			BusinessID: businessID,
			Name:       strings.TrimSpace(input.OutletName),
			Address:    strings.TrimSpace(input.OutletAddr),
			CreatedAt:  now,
		}},
	}
	if err := domain.Validate(created); err != nil {
		return domain.Business{}, err
	}
	if err := s.repository.Create(ctx, created); err != nil {
		return domain.Business{}, err
	}
	return created, nil
}

func (s *Service) List(ctx context.Context) ([]domain.Business, error) {
	return s.repository.List(ctx)
}

func (s *Service) ActiveID(ctx context.Context) (string, error) {
	return s.repository.ActiveID(ctx)
}

func (s *Service) SelectActive(ctx context.Context, id string) error {
	if strings.TrimSpace(id) == "" {
		return errors.New("pilih usaha terlebih dahulu")
	}

	businesses, err := s.repository.List(ctx)
	if err != nil {
		return err
	}
	for _, item := range businesses {
		if item.ID == id {
			return s.repository.SetActive(ctx, id)
		}
	}
	return errors.New("usaha tidak ditemukan")
}

func newID() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(value[:]), nil
}
