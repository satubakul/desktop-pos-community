package app

import (
	"context"
	"fmt"
	"time"

	"github.com/satubakul/desktop-pos-community/internal/adapters/fyneui"
	"github.com/satubakul/desktop-pos-community/internal/adapters/sqlite"
	businessapp "github.com/satubakul/desktop-pos-community/internal/application/business"
	catalogapp "github.com/satubakul/desktop-pos-community/internal/application/catalog"
	salesapp "github.com/satubakul/desktop-pos-community/internal/application/sales"
	stockapp "github.com/satubakul/desktop-pos-community/internal/application/stock"
)

func Run() error {
	// Single instance first, before touching the database: a second
	// click signals the running window and exits instead of opening a
	// duplicate window and database handle.
	var desktop *fyneui.App
	_, release, first := acquireSingleInstance(singleInstanceAddr, func() {
		if desktop != nil {
			desktop.Focus()
		}
	})
	if !first {
		fmt.Println("Kasir Kita sudah berjalan.")
		return nil
	}
	defer release()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	store, err := sqlite.OpenDefault(ctx)
	if err != nil {
		return fmt.Errorf("start local database: %w", err)
	}
	defer store.Close()

	businesses := businessapp.NewService(store)
	catalog := catalogapp.NewService(store)
	sales := salesapp.NewService(store)
	stock := stockapp.NewService(store)
	desktop = fyneui.New(businesses, catalog, sales, stock)
	desktop.Run()
	return nil
}
